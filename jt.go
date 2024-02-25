package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Colours
var colourCodes = map[string]string{
	"black":   "30",
	"red":     "31",
	"green":   "32",
	"yellow":  "33",
	"blue":    "34",
	"purple":  "35",
	"magenta": "35",
	"cyan":    "36",
	"white":   "37",
	"none":    "0",
}

// Options for the json print
type Options struct {
	Style         int
	NullChar      string
	Quote         string
	Equals        string
	Width         int
	KeyColour     string
	KeyBold       bool
	KeyInverted   bool
	ValueColour   string
	ValueBold     bool
	ValueInverted bool
	OtherColour   string
	OtherBold     bool
	OtherInverted bool
	StyleOther    MultiStringValue
}

type Style struct {
	Start    string
	Mid      string
	End      string
	Continue string
	None     string
	Array    string
	Null     string
}

var (
	debug = flag.Bool("debug", false, "debug mode")
)

type MultiStringValue []string

func (m *MultiStringValue) String() string {
	return fmt.Sprintf("%v", *m)
}

func (m *MultiStringValue) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	// Creat options struct for parser
	var options Options

	// Define flags
	flag.IntVar(&options.Style, "style", 0, "line style")
	flag.StringVar(&options.NullChar, "null-char", "", "value for empty array/dictionaries")
	flag.StringVar(&options.Quote, "quote", `"`, "quoting character(s)")
	flag.StringVar(&options.Equals, "equals", ":", "equals character(s)")
	flag.IntVar(&options.Width, "width", 1, "width of the tree")
	flag.StringVar(&options.KeyColour, "key-colour", "none", "key colour {black, red, green, yellow, blue, purple, magenta, cyan, white}")
	flag.BoolVar(&options.KeyBold, "key-bold", false, "bold font")
	flag.BoolVar(&options.KeyInverted, "key-inverted", false, "inverted font")
	flag.StringVar(&options.ValueColour, "value-colour", "none", "value colour {black, red, green, yellow, blue, purple, magenta, cyan, white}")
	flag.BoolVar(&options.ValueBold, "value-bold", false, "bolt font")
	flag.BoolVar(&options.ValueInverted, "value-inverted", false, "inverted font")
	flag.StringVar(&options.OtherColour, "other-colour", "none", "other colour {black, red, green, yellow, blue, purple, magenta, cyan, white}")
	flag.BoolVar(&options.OtherBold, "other-bold", false, "bold font")
	flag.BoolVar(&options.OtherInverted, "other-inverted", false, "inverted font")

	// Hidden
	flag.Var(&options.StyleOther, "s", "test style")

	// Custom help usage
	flag.Usage = func() {
		flagSet := flag.CommandLine
		fmt.Printf("Usage: %s [optional] [positional]\n\n", filepath.Base(flagSet.Name()))

		//goland:noinspection GoPrintFunctions
		fmt.Println("Print a json as a tree\n")

		fmt.Println("optional arguments:")
		order := map[string][]string{
			"general style": {"style", "null-char", "quote", "equals", "width"},
			"key style":     {"key-colour", "key-bold", "key-inverted"},
			"value style":   {"value-colour", "value-bold", "value-inverted"},
			"other style":   {"other-colour", "other-bold", "other-inverted"},
		}
		for key, values := range order {
			fmt.Println(key + ":")
			for _, value := range values {
				flagOption := flagSet.Lookup(value)
				fmt.Printf("    --%s", flagOption.Name)
				if flagOption.Value.String() == "0" || flagOption.Value.String() == "" {
					fmt.Printf("=%s", "None")
				} else {
					fmt.Printf("=%s", flagOption.Value)
				}
				fmt.Println()
				fmt.Printf("        %s ", flagOption.Usage)
				if flagOption.DefValue != "0" && flagOption.DefValue != "" && flagOption.DefValue != "false" {
					fmt.Printf("(default %s)", flagOption.DefValue)
				}
				fmt.Println()
			}
		}
		fmt.Println("\npositional arguments:\n\tjson/file")
		fmt.Println("\nstdin arguments:\n\tjson")
	}
	flag.Parse()

	// Options validation
	if options.Width < 0 || options.Width > 10 {
		fmt.Println("ERROR: Width out of range")
		flag.Usage()
		os.Exit(255)
	}
	if !isValidColour(options.KeyColour) {
		fmt.Println("ERROR: inValid color:", options.KeyColour)
		flag.Usage()
		os.Exit(255)
	}
	if !isValidColour(options.ValueColour) {
		fmt.Println("ERROR: inValid color:", options.ValueColour)
		flag.Usage()
		os.Exit(255)
	}
	if !isValidColour(options.OtherColour) {
		fmt.Println("ERROR: inValid color:", options.OtherColour)
		flag.Usage()
		os.Exit(255)
	}

	// Print options
	if *debug {
		fmt.Printf("%+v\n", options)
	}

	positionalArgs := flag.Args()
	if *debug {
		fmt.Println("Positional arguments:", positionalArgs)
	}

	stat, _ := os.Stdin.Stat()

	var stdin string

	// Check if stdin is a pipe and has data
	if (stat.Mode() & os.ModeNamedPipe) != 0 {
		// To create dynamic array
		scanner := bufio.NewScanner(os.Stdin)
		for {
			// Scans a line from Stdin(Console)
			scanner.Scan()
			// Holds the string that scanned
			text := scanner.Text()
			if len(text) != 0 {
				stdin += text
			} else {
				break
			}

		}
		// Parse the JSON data
		var data map[string]interface{}

		if err := json.Unmarshal([]byte(stdin), &data); err != nil {
			var arrData interface{}

			// Fail over to array
			err := json.Unmarshal([]byte(stdin), &arrData)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
			}
			// Call the function to print JSON data (with array)
			printJson("", arrData, options)
		} else {
			// Call the function to print JSON data
			printJson("", data, options)
		}

	}

	// Process positional arguments
	for _, value := range positionalArgs {
		var data map[string]interface{}

		validLoadFile, loadFilePath := validFilePath(value)
		if validLoadFile {
			jsonData, err := ioutil.ReadFile(loadFilePath)
			if err != nil {
				log.Fatal(err)
			}
			value = string(jsonData)
		}

		// Parse the JSON data
		if err := json.Unmarshal([]byte(value), &data); err != nil {
			var arrData interface{}

			// Fail over to array
			err := json.Unmarshal([]byte(value), &arrData)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				continue
			}
			// Call the function to print JSON data (with array)
			printJson("", arrData, options)
		} else {
			// Call the function to print JSON data
			printJson("", data, options)
		}
	}
}

// Get the symbol to appear before the line item
func symbol(i int, l int, mid string, end string) string {
	if i+1 != l {
		return mid
	} else {
		return end // Last in list
	}
}

// Get the nextSymbol to appear before the line item
func nextSymbol(i int, l int, cont string, none string) string {
	if i+1 != l {
		return cont
	} else {
		return none //Last in list
	}
}

// getStyle of the json printout
func getStyle(style int) Style {
	symbolsSets := [][]string{
		{"├─", "└─", "│ ", "  ", "─", "┐ ", "┘"},      //0
		{"┠─", "┖─", "┃ ", "  ", "─", "┒ ", "┘"},      //1
		{"┣━", "┗━", "┃ ", "  ", "━", "┓ ", "┛"},      //2
		{"╟─", "╙─", "║ ", "  ", "─", "╖ ", "┘"},      //3
		{"╠═", "╚═", "║ ", "  ", "═", "╗ ", "╝"},      //4
		{"├─", "╰─", "│ ", "  ", "─", "╮ ", "╯"},      //5
		{"╏╺", "┗╺", "╏ ", "  ", "╺", "┓ ", "╸╸╸╸"},   //6
		{"┡━", "┗━", "│ ", "  ", "━", "┒ ", "╾╶╶"},    //7
		{"┣━", "┺━", "┃ ", "  ", "━", "╅╴", "╾╶╶"},    //8
		{"▙▄", "▙▄", "▌ ", "  ", "▄", "▖ ", "▊▋▌▍▎▏"}, //9
		{"▕╲", " ╲", "▕ ", "  ", "▁", "▁ ", "╳╳╳╳╲"},  //10
	}

	if style < 0 || style >= len(symbolsSets) {
		// Default style if index is out of range
		style = 0
	}

	symbols := symbolsSets[style]

	return Style{
		Mid:      symbols[0],
		End:      symbols[1],
		Continue: symbols[2],
		None:     symbols[3],
		Array:    symbols[4],
		Start:    symbols[5],
		Null:     symbols[6],
	}
}

// printJson Recursively print json with style options
func printJson(indent string, data interface{}, options Options) {

	style := getStyle(options.Style)
	start := style.Start
	mid := style.Mid
	end := style.End
	cont := style.Continue + strings.Repeat(" ", options.Width)
	none := style.None + strings.Repeat(" ", options.Width)
	array := strings.Repeat(style.Array, options.Width) + style.Start
	null := style.Null

	if len(options.NullChar) > 0 {
		null = options.NullChar
	}

	// Check the type of the data
	if len(indent) == 0 {
		fmt.Println(Colour(start, options.OtherColour, options.OtherBold, options.OtherInverted))
	}
	switch assertedValue := data.(type) {
	case map[string]interface{}:
		length := len(assertedValue)
		index := 0
		if length == 0 {
			fmt.Printf("%s%s%s\n",
				Colour(indent, options.OtherColour, options.OtherBold, options.OtherInverted),
				Colour(end, options.OtherColour, options.OtherBold, options.OtherInverted),
				Colour(null, options.OtherColour, options.OtherBold, options.OtherInverted))
		}
		for key, value := range assertedValue {
			fmt.Printf("%s%s",
				Colour(indent+symbol(index, length, mid, end), options.OtherColour, options.OtherBold, options.OtherInverted),
				Colour(key, options.KeyColour, options.KeyBold, options.KeyInverted))
			switch value.(type) {
			case map[string]interface{}:
				fmt.Println()
				printJson(indent+nextSymbol(index, length, cont, none), value, options)
			case []interface{}:
				fmt.Println()
				printJson(indent+nextSymbol(index, length, cont, none), value, options)
			case string:
				fmt.Printf(" %s %s%s%s\n",
					Colour(options.Equals, options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(options.Quote, options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(fmt.Sprintf("%s", value), options.ValueColour, options.ValueBold, options.ValueInverted),
					Colour(options.Quote, options.OtherColour, options.OtherBold, options.OtherInverted))
			default:
				fmt.Printf(" %s %s\n",
					Colour(options.Equals, options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(fmt.Sprintf("%v", value), options.ValueColour, options.ValueBold, options.ValueInverted))
			}
			index++
		}
	case []interface{}:
		length := len(assertedValue)
		if length == 0 {
			fmt.Printf("%s%s%s\n",
				Colour(indent, options.OtherColour, options.OtherBold, options.OtherInverted),
				Colour(end, options.OtherColour, options.OtherBold, options.OtherInverted),
				Colour(null, options.OtherColour, options.OtherBold, options.OtherInverted))
		}
		for index, value := range assertedValue {
			switch value.(type) {
			case map[string]interface{}:
				fmt.Printf("%s%s%s\n",
					Colour(indent, options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(symbol(index, length, mid, end), options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(array, options.OtherColour, options.OtherBold, options.OtherInverted))
				printJson(indent+nextSymbol(index, length, cont, none), value, options)
			case []interface{}:
				fmt.Printf("%s%s%s\n",
					Colour(indent, options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(symbol(index, length, mid, end), options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(array, options.OtherColour, options.OtherBold, options.OtherInverted))
				printJson(indent+nextSymbol(index, length, cont, none), value, options)
			case string:
				fmt.Printf("%s %s%s%s\n",
					Colour(indent+symbol(index, length, mid, end), options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(options.Quote, options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(fmt.Sprintf("%s", value), options.ValueColour, options.ValueBold, options.ValueInverted),
					Colour(options.Quote, options.OtherColour, options.OtherBold, options.OtherInverted))
			default:
				fmt.Printf("%s %s\n",
					Colour(indent+symbol(index, length, mid, end), options.OtherColour, options.OtherBold, options.OtherInverted),
					Colour(fmt.Sprintf("%v", value), options.ValueColour, options.ValueBold, options.ValueInverted))
			}
		}
	}
}

// Colour returns the coloured version of the text if colour is supported and output is a terminal,
// otherwise returns the original text.
func Colour(text, colour string, bold bool, inverted bool) string {
	if supportsColour() && isTerminal() {
		boldFormat := "0"
		if bold {
			boldFormat = "1"
		}
		invertedFormat := ""
		if inverted {
			invertedFormat = ";7"
		}

		if code, ok := colourCodes[colour]; !ok {
			return fmt.Sprintf("\033[%s%sm%s\033[0m", boldFormat, invertedFormat, text)
		} else {
			return fmt.Sprintf("\033[%s%s;%sm%s\033[0m", boldFormat, invertedFormat, code, text)
		}

	}
	return text
}

// supportsColour checks if the terminal supports colour output.
func supportsColour() bool {
	return os.Getenv("TERM") != "dumb" && os.Getenv("COLORTERM") != "nocolour"
}

// isTerminal checks if the output is a terminal.
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// validFilePath Validates that a complete path exists and is a file at the pwd and the exec and returns the valid path
func validFilePath(path string) (bool, string) {
	info, err := os.Stat(path)
	if err != nil {
		// If os.Stat returns an error, the file or directory doesn't exist
		execPath, err := os.Executable()
		if err != nil {
			return false, path
		}
		execDir := filepath.Dir(execPath)
		filePath := filepath.Join(execDir, path)

		// Check if file exists with exec
		if info, err := os.Stat(filePath); err == nil {
			if !info.IsDir() {
				return true, filePath
			} else {
				return false, filePath
			}
		}
		return false, path
	}

	return !info.IsDir(), path
}

// isValidColor checks for valid colour in options
func isValidColour(color string) bool {
	_, ok := colourCodes[color]
	return ok
}
