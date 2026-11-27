import { createContext, useContext, useEffect } from 'react'

/** AdminLayout provides this; pages report whether they hold unsaved edits. */
export const UnsavedContext = createContext<{ setDirty: (dirty: boolean) => void }>({ setDirty: () => {} })

export function useUnsavedChanges(dirty: boolean): void {
  const { setDirty } = useContext(UnsavedContext)
  useEffect(() => {
    setDirty(dirty)
    return () => setDirty(false)
  }, [dirty, setDirty])
}

export const UNSAVED_MESSAGE = 'Bạn có thay đổi chưa lưu. Rời khỏi trang sẽ mất các thay đổi này. Vẫn rời đi?'
