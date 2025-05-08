package errx

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Output formats for CLI errors
type OutputFormat string

const (
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
)

// DisplayMode controls which elements of the error are displayed
type DisplayMode string

const (
	// DisplayModeSimple shows only the error message in a minimal format
	DisplayModeSimple DisplayMode = "simple"
	// DisplayModeNormal shows error message, code and type
	DisplayModeNormal DisplayMode = "normal"
	// DisplayModeDetailed shows all error information including details and cause chain
	DisplayModeDetailed DisplayMode = "detailed"
	// DisplayModeCustom allows for custom configuration of displayed fields
	DisplayModeCustom DisplayMode = "custom"
)

// CLIOptions configures how errors are displayed in CLI applications
type CLIOptions struct {
	// Format determines how errors are displayed (text or JSON)
	Format OutputFormat
	// DisplayMode controls which elements to display
	DisplayMode DisplayMode
	// ShowCode determines if error code is shown
	ShowCode bool
	// ShowType determines if error type is shown
	ShowType bool
	// ShowDetails determines if details are shown
	ShowDetails bool
	// ShowCause determines if the error cause chain is displayed
	ShowCause bool
	// ExitOnError determines if the program should exit on error
	ExitOnError bool
	// UseColors determines if colors should be used in text output
	UseColors bool
	// ExitFunc is the function called to exit the program
	ExitFunc func(int)
}

// DefaultCLIOptions returns the default options for CLI error handling
func DefaultCLIOptions() CLIOptions {
	return CLIOptions{
		Format:      OutputFormatText,
		DisplayMode: DisplayModeNormal,
		ShowCode:    true,
		ShowType:    true,
		ShowDetails: true,
		ShowCause:   true,
		ExitOnError: true,
		UseColors:   true,
		ExitFunc:    os.Exit,
	}
}

// SimpleCLIOptions returns minimalist options that only show the error message
func SimpleCLIOptions() CLIOptions {
	options := DefaultCLIOptions()
	options.DisplayMode = DisplayModeSimple
	return options
}

// DetailedCLIOptions returns options that show all error information
func DetailedCLIOptions() CLIOptions {
	options := DefaultCLIOptions()
	options.DisplayMode = DisplayModeDetailed
	return options
}

// applyDisplayMode updates the options based on the display mode
func (o *CLIOptions) applyDisplayMode() {
	switch o.DisplayMode {
	case DisplayModeSimple:
		o.ShowCode = false
		o.ShowType = false
		o.ShowDetails = false
		o.ShowCause = false
	case DisplayModeNormal:
		o.ShowCode = true
		o.ShowType = true
		o.ShowDetails = false
		o.ShowCause = false
	case DisplayModeDetailed:
		o.ShowCode = true
		o.ShowType = true
		o.ShowDetails = true
		o.ShowCause = true
	case DisplayModeCustom:
		// Use the explicit settings
	}
}

// CLI handles errors for command line applications
type CLI struct {
	options CLIOptions
}

// NewCLI creates a new CLI error handler with the given options
func NewCLI(options CLIOptions) *CLI {
	// Apply display mode settings
	options.applyDisplayMode()
	return &CLI{options: options}
}

// HandleCommandError wraps a Cobra command's RunE function to provide consistent error handling
func (c *CLI) HandleCommandError(runFn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := runFn(cmd, args)
		if err != nil {
			c.HandleError(err)
			return nil // Already handled the error
		}
		return nil
	}
}

// HandleError formats and outputs an error according to the CLI options
func (c *CLI) HandleError(err error) {
	if err == nil {
		return
	}

	exitCode := 1

	// Check if it's our Error type
	var xerr *Error
	if errors.As(err, &xerr) {
		// Set exit code based on error type
		switch xerr.Type {
		case TypeValidation:
			exitCode = 2
		case TypeAuthorization:
			exitCode = 3
		case TypeNotFound:
			exitCode = 4
		case TypeInternal:
			exitCode = 5
		}

		// Output the error based on format
		if c.options.Format == OutputFormatJSON {
			c.outputJSON(xerr)
		} else {
			c.outputBeautifulText(xerr)
		}
	} else {
		// Handle standard errors
		if c.options.Format == OutputFormatJSON {
			c.outputJSON(&Error{
				Code:    "UNKNOWN_ERROR",
				Type:    TypeInternal,
				Message: err.Error(),
			})
		} else {
			// Create a simple error display for non-errx errors
			genericErr := &Error{
				Code:    "UNKNOWN_ERROR",
				Type:    TypeInternal,
				Message: err.Error(),
			}
			c.outputBeautifulText(genericErr)
		}
	}

	if c.options.ExitOnError {
		c.options.ExitFunc(exitCode)
	}
}

// outputJSON outputs an error as JSON
func (c *CLI) outputJSON(err *Error) {
	output := map[string]interface{}{
		"error": map[string]interface{}{
			"message": err.Message,
		},
	}

	// Only include fields based on options
	if c.options.ShowCode {
		output["error"].(map[string]interface{})["code"] = err.Code
	}

	if c.options.ShowType {
		output["error"].(map[string]interface{})["type"] = err.Type
	}

	if c.options.ShowDetails && err.Details != nil && len(err.Details) > 0 {
		output["error"].(map[string]interface{})["details"] = err.Details
	}

	jsonBytes, _ := json.MarshalIndent(output, "", "  ")
	fmt.Fprintln(os.Stderr, string(jsonBytes))
}

// outputBeautifulText outputs an error as beautifully formatted text
func (c *CLI) outputBeautifulText(err *Error) {
	// Initialize colors
	var errorColor, codeColor, typeColor, detailKeyColor, messageColor, headerColor, lineColor *color.Color

	if c.options.UseColors {
		errorColor = color.New(color.FgHiRed, color.Bold)
		codeColor = color.New(color.FgHiYellow)
		typeColor = color.New(color.FgHiCyan)
		detailKeyColor = color.New(color.FgHiGreen)
		messageColor = color.New(color.FgHiWhite)
		headerColor = color.New(color.FgHiMagenta, color.Bold)
		lineColor = color.New(color.FgHiBlue)
	} else {
		// Create no-op colors
		errorColor = color.New()
		codeColor = color.New()
		typeColor = color.New()
		detailKeyColor = color.New()
		messageColor = color.New()
		headerColor = color.New()
		lineColor = color.New()
		color.NoColor = true
	}

	// For simple mode, just show the error message
	if c.options.DisplayMode == DisplayModeSimple {
		errorColor.Fprintf(os.Stderr, "Error: ")
		messageColor.Fprintln(os.Stderr, err.Message)
		return
	}

	// Create horizontal line
	line := strings.Repeat("─", 60)

	// Print error header
	fmt.Fprintln(os.Stderr)
	lineColor.Fprintln(os.Stderr, line)
	errorColor.Fprintf(os.Stderr, " ERROR ")
	headerColor.Fprintf(os.Stderr, "❯ ")
	messageColor.Fprintln(os.Stderr, err.Message)
	lineColor.Fprintln(os.Stderr, line)

	// Print error info in a clean format
	fmt.Fprintln(os.Stderr)

	// Show code if enabled
	if c.options.ShowCode {
		headerColor.Fprintf(os.Stderr, "   CODE ❯ ")
		codeColor.Fprintln(os.Stderr, string(err.Code))
	}

	// Show type if enabled
	if c.options.ShowType {
		headerColor.Fprintf(os.Stderr, "   TYPE ❯ ")
		typeColor.Fprintln(os.Stderr, string(err.Type))
	}

	// Print details if enabled and available
	if c.options.ShowDetails && err.Details != nil && len(err.Details) > 0 {
		fmt.Fprintln(os.Stderr)
		headerColor.Fprintln(os.Stderr, " DETAILS")

		for k, v := range err.Details {
			detailKeyColor.Fprintf(os.Stderr, "   %s", k)
			fmt.Fprintf(os.Stderr, " ❯ %v\n", v)
		}
	}

	// Print cause chain if enabled and available
	if c.options.ShowCause && err.cause != nil {
		fmt.Fprintln(os.Stderr)
		headerColor.Fprintln(os.Stderr, "   CAUSE")

		cause := err.cause
		indent := "   "
		for cause != nil {
			fmt.Fprintf(os.Stderr, "%s❯ %s\n", indent, cause.Error())

			// Try to unwrap for the next iteration
			indent += "  "
			cause = errors.Unwrap(cause)
		}
	}

	fmt.Fprintln(os.Stderr)
	lineColor.Fprintln(os.Stderr, line)
	fmt.Fprintln(os.Stderr)
}

// WithCLI registers a CLI error handler with a Cobra command
func WithCLI(cmd *cobra.Command, options CLIOptions) *CLI {
	cli := NewCLI(options)

	// Store the original PersistentPreRunE if it exists
	originalPreRun := cmd.PersistentPreRunE

	// Add our error handling wrapper
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Run the original PreRun if it exists
		if originalPreRun != nil {
			if err := originalPreRun(cmd, args); err != nil {
				cli.HandleError(err)
				return nil
			}
		}
		return nil
	}

	// Also handle errors from RunE
	if cmd.RunE != nil {
		originalRunE := cmd.RunE
		cmd.RunE = cli.HandleCommandError(originalRunE)
	}

	return cli
}

/*
# CLI Error Handling

The errx package provides flexible error display for CLI applications, allowing you to control exactly
how much error information is shown to your users.

## Display Modes

The package offers different error display modes:

### Simple Mode

Shows only the error message, perfect for end users or when you need minimal output:



Error: User with ID 123 not found



### Normal Mode (Default)

Shows the error message, code, and type:



────────────────────────────────────────────────────────
ERROR ❯ User not found
────────────────────────────────────────────────────────

CODE ❯ USER_NOT_FOUND
TYPE ❯ NOT_FOUND
────────────────────────────────────────────────────────



### Detailed Mode

Shows everything including error details and cause chain:



────────────────────────────────────────────────────────
ERROR ❯ User not found
────────────────────────────────────────────────────────

CODE ❯ USER_NOT_FOUND
TYPE ❯ NOT_FOUND

DETAILS
user_id ❯ 123456
searched_in ❯ local database
timestamp ❯ 2025-05-08T14:30:45Z

CAUSE
❯ connection refused: database is down
❯   no route to host

────────────────────────────────────────────────────────



### Custom Mode

Explicitly control which fields to display:

```go
cli := errx.NewCLI(errx.CLIOptions{
    DisplayMode: errx.DisplayModeCustom,
    ShowCode:    false,
    ShowType:    true,
    ShowDetails: true,
    ShowCause:   false,
})



Usage Examples


Basic Command with Simple Errors

For user-friendly commands with minimal error output:


func NewUserCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "user [command]",
        Short: "User management",
    }

    // Create error handler with simple display mode
    cli := errx.NewCLI(errx.SimpleCLIOptions())

    getUserCmd := &cobra.Command{
        Use:   "get [id]",
        Short: "Get user details",
        Args:  cobra.ExactArgs(1),
        RunE:  cli.HandleCommandError(func(cmd *cobra.Command, args []string) error {
            // Command implementation
            if userNotFound {
                return errx.New("User not found", errx.TypeNotFound)
            }
            return nil
        }),
    }

    cmd.AddCommand(getUserCmd)
    return cmd
}



Verbose Command for Developers

For commands that should show detailed error information:


func NewDebugCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "debug [command]",
        Short: "Debug tools",
    }

    // Create error handler with detailed display mode
    cli := errx.NewCLI(errx.DetailedCLIOptions())

    testDbCmd := &cobra.Command{
        Use:   "test-db",
        Short: "Test database connection",
        RunE:  cli.HandleCommandError(func(cmd *cobra.Command, args []string) error {
            // Simulated DB connection error
            dbErr := errors.New("connection timeout")
            return errx.Wrap(dbErr, "Database connection failed", errx.TypeSystem).
                WithDetail("host", "db.example.com").
                WithDetail("port", 5432).
                WithDetail("timeout_sec", 30)
        }),
    }

    cmd.AddCommand(testDbCmd)
    return cmd
}



Dynamic Error Display Based on Verbosity

Allow users to control the error verbosity level:


func NewRootCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "myapp",
        Short: "My CLI application",
    }

    // Add verbosity flag
    cmd.PersistentFlags().CountP("verbose", "v", "Increase error verbosity (can be used multiple times)")
    cmd.PersistentFlags().Bool("json", false, "Output errors as JSON")

    // Initialize and attach the error handler in PreRun
    cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
        // Get verbosity level
        verbose, _ := cmd.Flags().GetCount("verbose")
        jsonOutput, _ := cmd.Flags().GetBool("json")

        // Configure options based on verbosity
        options := errx.DefaultCLIOptions()

        if jsonOutput {
            options.Format = errx.OutputFormatJSON
        }

        // Set display mode based on verbosity level
        switch verbose {
        case 0:
            options.DisplayMode = errx.DisplayModeSimple
        case 1:
            options.DisplayMode = errx.DisplayModeNormal
        default:
            options.DisplayMode = errx.DisplayModeDetailed
        }

        // Attach error handler
        errx.WithCLI(cmd, options)
        return nil
    }

    // Add commands
    cmd.AddCommand(newUserCommand())

    return cmd
}


Usage examples:


$ myapp user get 123                 # Simple error output
$ myapp -v user get 123              # Normal error output with code and type
$ myapp -vv user get 123             # Detailed error output with everything
$ myapp --json user get 123          # JSON error output


With this approach, your CLI can adapt to different user needs - from simple error messages for end-users to detailed diagnostic information for developers and troubleshooting.
*/
