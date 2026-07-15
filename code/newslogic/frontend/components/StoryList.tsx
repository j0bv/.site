import { useQuery } from '@tanstack/react-query'
import { storyService, type Story } from '@was/api-client'
import { format } from 'date-fns'
import { Button } from '@was/ui'
import { Edit, Trash2, Eye, EyeOff } from 'lucide-react'

interface StoryListProps {
  onSelectStory?: (story: Story) => void
  onEditStory?: (story: Story) => void
}

/**
 * Component to display list of user's stories.
 */
export function StoryList({ onSelectStory, onEditStory }: StoryListProps) {
  const { data: storiesData, isLoading, refetch } = useQuery({
    queryKey: ['stories'],
    queryFn: () => storyService.getStories(),
  })

  const stories = storiesData?.stories || []

  const handleDelete = async (storyId: string) => {
    if (!confirm('Are you sure you want to delete this story?')) return

    try {
      await storyService.deleteStory(storyId)
      refetch()
    } catch (error) {
      console.error('Failed to delete story:', error)
    }
  }

  const handleTogglePublish = async (story: Story) => {
    try {
      if (story.status === 'published') {
        await storyService.unpublishStory(story.id)
      } else {
        await storyService.publishStory(story.id)
      }
      refetch()
    } catch (error) {
      console.error('Failed to toggle publish status:', error)
    }
  }

  if (isLoading) {
    return <div className="p-4 text-muted-foreground">Loading stories...</div>
  }

  if (stories.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <p className="text-muted-foreground">No stories yet</p>
        <p className="text-muted-foreground/60 mt-1 text-sm">Create your first story to get started</p>
      </div>
    )
  }

  return (
    <div className="divide-border divide-y">
      {stories.map((story) => (
        <div
          key={story.id}
          className="hover:bg-accent/50 p-4 transition-colors cursor-pointer"
          onClick={() => onSelectStory?.(story)}
        >
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1 min-w-0">
              <h3 className="font-semibold truncate">{story.title}</h3>
              <div className="text-muted-foreground mt-1 flex items-center gap-2 text-sm">
                <span className={`px-2 py-0.5 rounded text-xs ${story.status === 'published' ? 'bg-green-500/20 text-green-600' : 'bg-gray-500/20 text-gray-600'}`}>
                  {story.status}
                </span>
                <span>{format(new Date(story.updated_at), 'MMM d, yyyy')}</span>
              </div>
            </div>
            <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => onEditStory?.(story)}
                title="Edit"
              >
                <Edit className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => handleTogglePublish(story)}
                title={story.status === 'published' ? 'Unpublish' : 'Publish'}
              >
                {story.status === 'published' ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => handleDelete(story.id)}
                title="Delete"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

