package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/yjwong/lark-cli/internal/api"
	"github.com/yjwong/lark-cli/internal/output"
)

var slidesCmd = &cobra.Command{
	Use:   "slides",
	Short: "Lark Slides (presentations)",
	Long:  "Create, view, and edit Lark Slides presentations",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		validateScopeGroup("slides")
	},
}

// --- slides create ---

var (
	slidesCreateTitle       string
	slidesCreateFolderToken string
)

var slidesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new empty presentation",
	Long: `Create a new empty Lark Slides presentation.

If --folder is omitted, the presentation lands in the user's root drive.

Examples:
  lark slides create --title "Q4 Review"
  lark slides create --title "Roadmap" --folder fldcnxxxxx`,
	Run: func(cmd *cobra.Command, args []string) {
		if slidesCreateTitle == "" {
			output.Fatalf("VALIDATION_ERROR", "--title is required")
		}
		client := api.NewClient()
		p, err := client.CreateSlidesPresentation(slidesCreateTitle, slidesCreateFolderToken)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(p)
	},
}

// --- slides get ---

var slidesGetCmd = &cobra.Command{
	Use:   "get <presentation-id>",
	Short: "Get presentation metadata",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		data, err := client.GetSlidesPresentation(args[0])
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(data)
	},
}

// --- slides slide {create,delete} ---

var slideCmd = &cobra.Command{
	Use:   "slide",
	Short: "Create or delete individual slides",
}

var (
	slideCreateXML     string
	slideCreateXMLFile string
)

var slideCreateCmd = &cobra.Command{
	Use:   "create <presentation-id>",
	Short: "Append a new slide to a presentation via XML content",
	Long: `Append a slide using Lark's XML slide definition.

Provide the XML inline with --xml, or via a file with --xml-file (use "-" for stdin).

Examples:
  lark slides slide create pres_xxx --xml-file slide.xml
  cat slide.xml | lark slides slide create pres_xxx --xml-file -`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		xml := slideCreateXML
		if slideCreateXMLFile != "" {
			var data []byte
			var err error
			if slideCreateXMLFile == "-" {
				data, err = readAllStdin()
			} else {
				data, err = os.ReadFile(slideCreateXMLFile)
			}
			if err != nil {
				output.Fatal("FILE_ERROR", err)
			}
			xml = string(data)
		}
		if xml == "" {
			output.Fatalf("VALIDATION_ERROR", "--xml or --xml-file is required")
		}
		client := api.NewClient()
		data, err := client.CreateSlidesSlide(args[0], xml)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(data)
	},
}

var slideDeleteID string

var slideDeleteCmd = &cobra.Command{
	Use:   "delete <presentation-id>",
	Short: "Delete a slide by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if slideDeleteID == "" {
			output.Fatalf("VALIDATION_ERROR", "--slide-id is required")
		}
		client := api.NewClient()
		if err := client.DeleteSlidesSlide(args[0], slideDeleteID); err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"success": true, "slide_id": slideDeleteID})
	},
}

// --- slides media upload ---

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Upload media for use in slides/docs",
}

var (
	mediaParentType string
	mediaParentNode string
)

var mediaUploadCmd = &cobra.Command{
	Use:   "upload <file>",
	Short: "Upload a file to Drive and return its file_token",
	Long: `Upload an image or other media file to Lark Drive.

Returns a file_token that can be referenced in slide or document XML content.

Examples:
  lark slides media upload diagram.png
  lark slides media upload cover.jpg --parent-type slides_image`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := api.NewClient()
		token, err := client.UploadMedia(args[0], mediaParentType, mediaParentNode)
		if err != nil {
			output.Fatal("API_ERROR", err)
		}
		output.JSON(map[string]interface{}{"file_token": token})
	},
}

func readAllStdin() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

func init() {
	slidesCreateCmd.Flags().StringVar(&slidesCreateTitle, "title", "", "Presentation title (required)")
	slidesCreateCmd.Flags().StringVar(&slidesCreateFolderToken, "folder", "", "Destination folder_token")

	slideCreateCmd.Flags().StringVar(&slideCreateXML, "xml", "", "Slide XML content (inline)")
	slideCreateCmd.Flags().StringVar(&slideCreateXMLFile, "xml-file", "", "Slide XML content file path ('-' for stdin)")

	slideDeleteCmd.Flags().StringVar(&slideDeleteID, "slide-id", "", "Slide ID to delete (required)")

	mediaUploadCmd.Flags().StringVar(&mediaParentType, "parent-type", "slides_image", "Parent type: slides_image, docx_image, etc.")
	mediaUploadCmd.Flags().StringVar(&mediaParentNode, "parent-node", "", "Parent node token (optional)")

	slideCmd.AddCommand(slideCreateCmd)
	slideCmd.AddCommand(slideDeleteCmd)
	mediaCmd.AddCommand(mediaUploadCmd)

	slidesCmd.AddCommand(slidesCreateCmd)
	slidesCmd.AddCommand(slidesGetCmd)
	slidesCmd.AddCommand(slideCmd)
	slidesCmd.AddCommand(mediaCmd)
}
