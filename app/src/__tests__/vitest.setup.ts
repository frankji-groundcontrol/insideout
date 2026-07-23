// happy-dom@20 + Node 的组合下不再自动在全局注入 localStorage/sessionStorage，
// 而测试套件（store/组件/适配器）依赖它们。这里提供一个最小的内存版 Storage 兜底，
// 仅在缺失时安装，不覆盖 happy-dom 已提供的实现。文件名不匹配 *.{test,spec}.ts，
// 因此不会被当作测试用例收集。

class MemoryStorage implements Storage {
  private store = new Map<string, string>()

  get length(): number {
    return this.store.size
  }

  clear(): void {
    this.store.clear()
  }

  getItem(key: string): string | null {
    return this.store.has(key) ? (this.store.get(key) as string) : null
  }

  key(index: number): string | null {
    return Array.from(this.store.keys())[index] ?? null
  }

  removeItem(key: string): void {
    this.store.delete(key)
  }

  setItem(key: string, value: string): void {
    this.store.set(key, String(value))
  }
}

function isUsableStorage(value: unknown): value is Storage {
  return typeof (value as Storage | undefined)?.clear === 'function'
}

function ensureStorage(name: 'localStorage' | 'sessionStorage'): void {
  const g = globalThis as unknown as Record<string, unknown>
  // happy-dom@20 defines globalThis.localStorage as a non-null stub that's
  // missing methods like .clear() in some Node/happy-dom combinations, so a
  // plain `== null` check no longer detects the broken case.
  if (!isUsableStorage(g[name])) {
    const storage = new MemoryStorage()
    g[name] = storage
    // 同步挂到 window 上，便于通过 window.localStorage 访问的代码。
    const win = g.window as Record<string, unknown> | undefined
    if (win && !isUsableStorage(win[name])) {
      win[name] = storage
    }
  }
}

ensureStorage('localStorage')
ensureStorage('sessionStorage')
