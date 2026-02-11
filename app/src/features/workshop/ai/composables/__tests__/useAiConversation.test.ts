import { ref } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AiRateLimitError, AiServiceUnavailableError } from "@/types";

const replyMock = vi.hoisted(() => vi.fn());
const getConversationMock = vi.hoisted(() => vi.fn());
const clearConversationMock = vi.hoisted(() => vi.fn());
const enqueueInsertMock = vi.hoisted(() => vi.fn());

vi.mock("@/services/registry", () => ({
	requireServices: () => ({
		ai: {
			reply: replyMock,
			getConversation: getConversationMock,
			clearConversation: clearConversationMock,
		},
	}),
}));

vi.mock("@/stores/editor", () => ({
	useEditorStore: () => ({
		enqueueInsert: enqueueInsertMock,
	}),
}));

import { useAiConversation } from "@/features/workshop/ai/composables/useAiConversation";

describe("useAiConversation", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		getConversationMock.mockResolvedValue(undefined);
		clearConversationMock.mockResolvedValue(undefined);
	});

	it("does not send request when nodeId is empty", async () => {
		const nodeId = ref("");
		const conversation = useAiConversation({ nodeId });

		conversation.draft.value = "hello";
		await conversation.sendMessage();

		expect(replyMock).not.toHaveBeenCalled();
		expect(conversation.messages.value).toHaveLength(0);
		expect(conversation.isThinking.value).toBe(false);
	});

	it("resets thinking state when ai request fails", async () => {
		replyMock.mockRejectedValueOnce(new Error("boom"));
		const nodeId = ref("node_1");
		const conversation = useAiConversation({ nodeId });

		await Promise.resolve();

		conversation.draft.value = "hello";
		await conversation.sendMessage();

		expect(replyMock).toHaveBeenCalledWith("node_1", "hello");
		expect(conversation.isThinking.value).toBe(false);
		expect(conversation.messages.value[0]?.role).toBe("user");
		expect(conversation.messages.value[1]?.role).toBe("assistant");
		expect(conversation.messages.value[1]?.content).toContain("暂时不可用");
	});

	it("loads saved conversation history when node is selected", async () => {
		getConversationMock.mockResolvedValueOnce({
			nodeId: "node_1",
			messages: [
				{
					id: "saved_1",
					role: "assistant",
					content: "saved reply",
					timestamp: "2026-02-11T00:00:00.000Z",
				},
			],
		});

		const nodeId = ref("node_1");
		const conversation = useAiConversation({ nodeId });

		await Promise.resolve();

		expect(getConversationMock).toHaveBeenCalledWith("node_1");
		expect(conversation.messages.value).toHaveLength(1);
		expect(conversation.messages.value[0]?.content).toBe("saved reply");
	});

	it("starts a new conversation by clearing persisted history", async () => {
		getConversationMock.mockResolvedValueOnce({
			nodeId: "node_1",
			messages: [
				{
					id: "saved_1",
					role: "assistant",
					content: "saved reply",
					timestamp: "2026-02-11T00:00:00.000Z",
				},
			],
		});

		const nodeId = ref("node_1");
		const conversation = useAiConversation({ nodeId });

		await Promise.resolve();
		conversation.draft.value = "old draft";
		await conversation.startNewConversation();

		expect(clearConversationMock).toHaveBeenCalledWith("node_1");
		expect(conversation.messages.value).toHaveLength(0);
		expect(conversation.draft.value).toBe("");
	});

	it("shows retry countdown when ai is rate-limited", async () => {
		replyMock.mockRejectedValueOnce(new AiRateLimitError("too many", 3));
		const nodeId = ref("node_1");
		const conversation = useAiConversation({ nodeId });

		await Promise.resolve();
		conversation.draft.value = "hello";
		await conversation.sendMessage();

		expect(conversation.isRateLimited.value).toBe(true);
		expect(conversation.retryCountdown.value).toBe(3);
		expect(conversation.messages.value[1]?.content).toContain("3");
	});

	it("shows service unavailable countdown when circuit is open", async () => {
		replyMock.mockRejectedValueOnce(
			new AiServiceUnavailableError("circuit open", 5, "open"),
		);
		const nodeId = ref("node_1");
		const conversation = useAiConversation({ nodeId });

		await Promise.resolve();
		conversation.draft.value = "hello";
		await conversation.sendMessage();

		expect(conversation.isRateLimited.value).toBe(true);
		expect(conversation.retryCountdown.value).toBe(5);
		expect(conversation.messages.value[1]?.content).toContain("暂时不可用");
	});
});
