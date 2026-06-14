import { computed, nextTick, type Ref, ref, watch } from "vue";
import { useServices } from "@/composables/useServices";
import { useEditorStore } from "@/stores/editor";
import {
	AiRateLimitError,
	AiServiceUnavailableError,
	type AiMessage,
} from "@/types";

interface UseAiConversationOptions {
	nodeId: Ref<string>;
}

export function useAiConversation(options: UseAiConversationOptions) {
	const editorStore = useEditorStore();

	const draft = ref("");
	const isThinking = ref(false);
	const listEl = ref<HTMLElement | null>(null);
	const requestToken = ref(0);
	const messages = ref<AiMessage[]>([]);
	const retryCountdown = ref(0);
	const isRateLimited = ref(false);
	const adoptedIds = ref<Set<string>>(new Set());
	let retryTimer: ReturnType<typeof setInterval> | null = null;
	const hasNode = computed(() => options.nodeId.value.trim().length > 0);

	const canSend = computed(
		() =>
			hasNode.value &&
			draft.value.trim().length > 0 &&
			!isThinking.value &&
			!isRateLimited.value,
	);

	function clearRetryTimer() {
		if (retryTimer) {
			clearInterval(retryTimer);
			retryTimer = null;
		}
	}

	function startRetryCountdown(seconds: number) {
		clearRetryTimer();
		retryCountdown.value = seconds;
		isRateLimited.value = true;
		retryTimer = setInterval(() => {
			if (retryCountdown.value <= 1) {
				clearRetryTimer();
				retryCountdown.value = 0;
				isRateLimited.value = false;
				return;
			}
			retryCountdown.value -= 1;
		}, 1000);
	}

	function buildUserMessage(content: string): AiMessage {
		return {
			id: `local_user_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
			role: "user",
			content,
			timestamp: new Date().toISOString(),
		};
	}

	function buildErrorMessage(): AiMessage {
		return {
			id: `local_error_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
			role: "assistant",
			content: "AI 服务暂时不可用，请稍后再试。",
			timestamp: new Date().toISOString(),
		};
	}

	function buildRateLimitMessage(seconds: number): AiMessage {
		return {
			id: `local_error_rate_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
			role: "assistant",
			content: `请求过于频繁，请在 ${seconds} 秒后重试。`,
			timestamp: new Date().toISOString(),
		};
	}

	function buildServiceUnavailableMessage(seconds: number): AiMessage {
		return {
			id: `local_error_unavailable_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
			role: "assistant",
			content: `AI 服务暂时不可用，请在 ${seconds} 秒后重试。`,
			timestamp: new Date().toISOString(),
		};
	}

	function formatTime(iso: string): string {
		const date = new Date(iso);
		if (Number.isNaN(date.getTime())) {
			return "--:--";
		}
		return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
	}

	async function scrollToBottom() {
		await nextTick();
		if (!listEl.value) {
			return;
		}
		listEl.value.scrollTop = listEl.value.scrollHeight;
	}

	function adoptMessage(message: AiMessage) {
		if (adoptedIds.value.has(message.id)) {
			return;
		}

		editorStore.enqueueInsert(message.content);
		const next = new Set(adoptedIds.value);
		next.add(message.id);
		adoptedIds.value = next;
	}

	async function sendMessage() {
		const content = draft.value.trim();
		if (!content || isThinking.value) {
			return;
		}

		if (!hasNode.value) {
			return;
		}

		const userMessage = buildUserMessage(content);
		messages.value.push(userMessage);
		draft.value = "";
		isThinking.value = true;
		await scrollToBottom();

		const token = requestToken.value + 1;
		requestToken.value = token;
		const nodeAtRequest = options.nodeId.value;

		try {
			const reply = await useServices().ai.reply(nodeAtRequest, content);
			if (
				token !== requestToken.value ||
				nodeAtRequest !== options.nodeId.value
			) {
				return;
			}
			messages.value.push(reply);
		} catch (error) {
			if (
				token === requestToken.value &&
				nodeAtRequest === options.nodeId.value
			) {
				if (error instanceof AiRateLimitError) {
					startRetryCountdown(error.retryAfterSeconds);
					messages.value.push(buildRateLimitMessage(error.retryAfterSeconds));
				} else if (error instanceof AiServiceUnavailableError) {
					startRetryCountdown(error.retryAfterSeconds);
					messages.value.push(buildServiceUnavailableMessage(error.retryAfterSeconds));
				} else {
					messages.value.push(buildErrorMessage());
				}
			}
		} finally {
			if (token === requestToken.value) {
				isThinking.value = false;
			}
			await scrollToBottom();
		}
	}

	async function loadConversation(nodeId: string, token: number) {
		if (!nodeId.trim()) {
			messages.value = [];
			return;
		}

		try {
			const conversation = await useServices().ai.getConversation(nodeId);
			if (token !== requestToken.value || nodeId !== options.nodeId.value) {
				return;
			}
			messages.value = conversation?.messages ?? [];
		} catch {
			if (token !== requestToken.value || nodeId !== options.nodeId.value) {
				return;
			}
			messages.value = [];
		}
	}

	async function startNewConversation() {
		const nodeId = options.nodeId.value.trim();
		if (!nodeId) {
			return;
		}

		requestToken.value += 1;
		draft.value = "";
		isThinking.value = false;
		clearRetryTimer();
		retryCountdown.value = 0;
		isRateLimited.value = false;
		messages.value = [];
		adoptedIds.value = new Set();

		await useServices().ai.clearConversation(nodeId);
		await scrollToBottom();
	}

	watch(
		() => options.nodeId.value,
		async nodeId => {
			const token = requestToken.value + 1;
			requestToken.value = token;
			draft.value = "";
			isThinking.value = false;
			clearRetryTimer();
			retryCountdown.value = 0;
			isRateLimited.value = false;
			adoptedIds.value = new Set();
			await loadConversation(nodeId, token);
			await scrollToBottom();
		},
		{ immediate: true },
	);

	watch(
		() => [messages.value.length, isThinking.value],
		async () => {
			await scrollToBottom();
		},
	);

	return {
		draft,
		isThinking,
		listEl,
		messages,
		retryCountdown,
		isRateLimited,
		adoptedIds,
		hasNode,
		canSend,
		formatTime,
		adoptMessage,
		sendMessage,
		startNewConversation,
	};
}
