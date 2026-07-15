import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import { useState } from 'react'
import { Button } from '@was/ui'
import { storyService, type Story } from '@was/api-client'

interface StoryEditorProps {
  story?: Story | null
  onSave?: (story: Story) => void
  onCancel?: () => void
}

/**
 * Rich text story editor using TipTap.
 */
export function StoryEditor({ story, onSave, onCancel }: StoryEditorProps) {
  const [title, setTitle] = useState(story?.title || '')
  const [isSaving, setIsSaving] = useState(false)
  const [status, setStatus] = useState<'draft' | 'published'>(story?.status || 'draft')

  const editor = useEditor({
    extensions: [
      StarterKit,
      Placeholder.configure({
        placeholder: 'Start writing your story...',
      }),
    ],
    content: story?.content || '',
    editorProps: {
      attributes: {
        class: 'prose prose-lg max-w-none focus:outline-none min-h-[400px] p-4',
      },
    },
  })

  const handleSave = async () => {
    if (!editor || !title.trim()) return

    setIsSaving(true)
    try {
      const content = editor.getJSON()

      if (story) {
        // Update existing story
        const updated = await storyService.updateStory(story.id, {
          title,
          content,
          status,
        })
        onSave?.(updated)
      } else {
        // Create new story
        const created = await storyService.createStory({
          title,
          content,
          status,
        })
        onSave?.(created)
      }
    } catch (error) {
      console.error('Failed to save story:', error)
    } finally {
      setIsSaving(false)
    }
  }

  const handlePublish = async () => {
    if (!story) {
      await handleSave()
      return
    }

    setIsSaving(true)
    try {
      const published = await storyService.publishStory(story.id)
      setStatus('published')
      onSave?.(published)
    } catch (error) {
      console.error('Failed to publish story:', error)
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="border-border bg-card border-b p-4">
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Story title..."
          className="bg-background text-foreground w-full border-none text-2xl font-bold focus:outline-none"
        />
        <div className="mt-2 flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setStatus(status === 'draft' ? 'published' : 'draft')}
          >
            {status === 'draft' ? 'Draft' : 'Published'}
          </Button>
          <div className="flex-1" />
          {onCancel && (
            <Button variant="ghost" size="sm" onClick={onCancel}>
              Cancel
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={handleSave} disabled={isSaving || !title.trim()}>
            {isSaving ? 'Saving...' : 'Save'}
          </Button>
          {status === 'draft' && (
            <Button size="sm" onClick={handlePublish} disabled={isSaving || !title.trim()}>
              Publish
            </Button>
          )}
        </div>
      </div>

      {/* Editor */}
      <div className="flex-1 overflow-auto">
        <EditorContent editor={editor} />
      </div>
    </div>
  )
}

