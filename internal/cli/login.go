package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/diranged/claude-profile-go/internal/profile"
	"github.com/spf13/cobra"
)

// colorPresets defines the named ANSI 256-color codes offered during the
// interactive profile creation wizard. The first entry (Green) is the default
// when the user presses Enter without choosing.
var colorPresets = []struct {
	name string // human-readable name shown in the menu
	code int    // ANSI 256-color code (0-255)
}{
	{"Green (default)", 108},
	{"Blue", 33},
	{"Orange", 208},
	{"Pink", 204},
	{"Cyan", 51},
	{"Red", 196},
	{"Purple", 141},
	{"Yellow", 226},
}

// newCreateCmd builds the "create" subcommand, which walks the user through
// an interactive wizard to set up a new profile. The wizard:
//  1. Creates the profile's isolated config directory.
//  2. Offers to copy config files (CLAUDE.md, settings.json, etc.) from
//     the default ~/.claude directory.
//  3. Lets the user pick a banner/statusline color.
//  4. Configures the Claude Code statusline to display profile info.
//  5. Prints next-step instructions for authenticating the new profile.
func newCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <profile>",
		Short: "Create a new profile and launch Claude to authenticate",
		Long: `Creates a new isolated profile directory, walks through a setup wizard
(color selection, config bootstrap), then launches Claude so you can
authenticate however you prefer (OAuth, API key, Bedrock, etc.).

Example:
  claude-profile create work
  claude-profile create personal`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			p := profile.Load(name)

			if p.Exists() {
				return fmt.Errorf("profile %q already exists", name)
			}

			if err := p.EnsureDir(); err != nil {
				return fmt.Errorf("creating profile directory: %w", err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "\nCreating profile: %s\n", name)
			fmt.Fprintf(out, "Config directory: %s\n\n", p.ConfigDir)

			// Step 1: Bootstrap config files
			if err := offerBootstrap(p); err != nil {
				return err
			}

			// Step 2: Color selection
			color := pickColor()
			cfg := profile.DefaultConfig()
			cfg.Color = color
			if err := p.SaveConfig(cfg); err != nil {
				return fmt.Errorf("saving profile config: %w", err)
			}

			// Step 3: Configure statusline
			if err := configureStatusline(p); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to configure statusline: %s\n", err)
			}

			// Print next steps
			fmt.Fprintf(out, "\nProfile %q created!\n\n", name)
			fmt.Fprintf(out, "To authenticate, run claude-profile with your profile and use Claude's\n")
			fmt.Fprintf(out, "built-in auth commands:\n\n")
			fmt.Fprintf(out, "  Claude subscription (OAuth):\n")
			fmt.Fprintf(out, "    claude-profile -p %s\n", name)
			fmt.Fprintf(out, "    Then use /login inside Claude\n\n")
			fmt.Fprintf(out, "  Claude subscription (CLI):\n")
			fmt.Fprintf(out, "    claude-profile -p %s auth login\n", name)
			fmt.Fprintf(out, "    claude-profile -p %s auth login --sso\n\n", name)
			fmt.Fprintf(out, "  API key:\n")
			fmt.Fprintf(out, "    ANTHROPIC_API_KEY=sk-... claude-profile -p %s\n\n", name)
			fmt.Fprintf(out, "  AWS Bedrock:\n")
			fmt.Fprintf(out, "    CLAUDE_CODE_USE_BEDROCK=1 claude-profile -p %s\n\n", name)
			fmt.Fprintf(out, "  Google Vertex:\n")
			fmt.Fprintf(out, "    CLAUDE_CODE_USE_VERTEX=1 claude-profile -p %s\n\n", name)

			return nil
		},
	}
}

// pickColor presents an interactive menu of color presets plus a custom-code
// option, reads the user's choice from stdin, and returns the selected ANSI
// 256-color code. Invalid or empty input defaults to the first preset (green, 108).
func pickColor() int {
	fmt.Println("Pick a color for this profile's banner and statusline:")
	for i, preset := range colorPresets {
		color := fmt.Sprintf("\033[38;5;%dm", preset.code)
		reset := "\033[0m"
		fmt.Printf("  %s%d) %s (%d)%s\n", color, i+1, preset.name, preset.code, reset)
	}
	fmt.Printf("  %d) Custom (enter ANSI 256-color code)\n", len(colorPresets)+1)
	fmt.Printf("\nChoice [1]: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Default to first preset
	if input == "" {
		return colorPresets[0].code
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 {
		return colorPresets[0].code
	}

	// Preset selection
	if choice <= len(colorPresets) {
		return colorPresets[choice-1].code
	}

	// Custom color
	if choice == len(colorPresets)+1 {
		fmt.Print("Enter ANSI 256-color code (0-255): ")
		input, _ = reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if code, err := strconv.Atoi(input); err == nil && code >= 0 && code <= 255 {
			return code
		}
	}

	return colorPresets[0].code
}

// configureStatusline updates the profile's settings.json to use claude-profile's
// statusline wrapper. If the user already has a statusline command configured,
// the existing command is preserved by chaining it after our wrapper using the
// "-- <original>" syntax. If there is no existing statusline, we set ours as
// the sole statusline command. The statusline command is set with padding: 0
// and type: "command" to match Claude Code's expected format.
func configureStatusline(p *profile.Profile) error {
	// Find our own binary path for the statusline command
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding own binary: %w", err)
	}

	settingsPath := p.ConfigDir + "/settings.json"

	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	// Check for existing statusline command
	var origCmd string
	if sl, ok := settings["statusLine"].(map[string]interface{}); ok {
		if cmd, ok := sl["command"].(string); ok && cmd != "" {
			origCmd = cmd
		}
	}

	// Build our statusline command
	var statuslineCmd string
	if origCmd != "" {
		statuslineCmd = fmt.Sprintf("%s statusline -- %s", self, origCmd)
		fmt.Printf("Wrapping existing statusline: %s\n", origCmd)
	} else {
		statuslineCmd = fmt.Sprintf("%s statusline", self)
		fmt.Println("Configured profile statusline.")
	}

	settings["statusLine"] = map[string]interface{}{
		"type":    "command",
		"command": statuslineCmd,
		"padding": 0,
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	return os.WriteFile(settingsPath, out, 0644)
}

// offerBootstrap checks if the default ~/.claude directory has config files
// that could be copied into a new profile, and prompts the user.
func offerBootstrap(p *profile.Profile) error {
	files := profile.BootstrapFiles()
	if len(files) == 0 {
		return nil
	}

	fmt.Printf("Found config files in %s:\n", profile.DefaultConfigDir())
	for _, f := range files {
		fmt.Printf("  - %s\n", f)
	}
	fmt.Print("Copy these into the new profile? [Y/n] ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" || input == "y" || input == "yes" {
		if err := p.CopyBootstrapFiles(files); err != nil {
			return fmt.Errorf("copying config files: %w", err)
		}
		fmt.Printf("Copied %d file(s) into profile.\n", len(files))
	} else {
		fmt.Println("Skipped.")
	}

	return nil
}
