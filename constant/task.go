package constant

type TaskPlatform string

const (
	TaskPlatformSuno             TaskPlatform = "suno"
	TaskPlatformMidjourney                    = "mj"
	TaskPlatformGrsai                         = "grsai"
	TaskPlatformAsyncImageAli                 = "async_image_ali"
	TaskPlatformAsyncImageNewAPI              = "async_image_new_api"
)

const (
	SunoActionMusic  = "MUSIC"
	SunoActionLyrics = "LYRICS"

	TaskActionGenerate          = "generate"
	TaskActionTextGenerate      = "textGenerate"
	TaskActionFirstTailGenerate = "firstTailGenerate"
	TaskActionReferenceGenerate = "referenceGenerate"
	TaskActionRemix             = "remixGenerate"
	TaskActionImageGenerate     = "imageGenerate"
)

var SunoModel2Action = map[string]string{
	"suno_music":  SunoActionMusic,
	"suno_lyrics": SunoActionLyrics,
}
