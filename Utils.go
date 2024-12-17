package jw6_utils

import (
	"fmt"
	"strings"
	"time"
)

type Utils struct {
	LogLevel LogLevel
	// Define fields...
}

type LogLevel int

const (
	Trace LogLevel = iota
	Debug
	Info
	Warn
	Error
	Fatal
)

func LogLevelFromString(level string) LogLevel {
	switch level {
	case "Debug":
		return Debug
	case "Info":
		return Info
	case "Warn":
		return Warn
	case "Error":
		return Error
	case "Fatal":
		return Fatal
	default:
		fmt.Println("Default log level applied")
		return Info // or any default level you want to set
	}
}

// Method to get the label of the LogLevel constant
func (l LogLevel) String() string {
	labels := [...]string{"Trace", "Debug", "Info", "Warn", "Error", "Fatal"}
	if l < Trace || l > Fatal {
		return "Unknown"
	}
	return labels[l]
}

func (u *Utils) Log(class string, method string, level LogLevel, message string) {
	if level >= u.LogLevel {
		currentTime := time.Now()
		formattedTime := currentTime.Format("2006-01-02 15:04:05")
		if level <= Info {
			fmt.Print(formattedTime, " [", class, "][", method, "][", level.String(), "] - ", message, "\n")
		} else {
			// Color ANSI escape code
			red := "\033[31m"
			yellow := "\033[33m"
			green := "\033[32m"
			// Reset ANSI escape code (to reset color)
			reset := "\033[0m"

			if level == Info {
				fmt.Print(formattedTime, " [", class, "][", method, "][", level.String(), "] - ", green, message, reset, "\n")
			} else if level == Warn {
				fmt.Print(formattedTime, " [", class, "][", method, "][", level.String(), "] - ", yellow, message, reset, "\n")
			} else if level <= Error {
				fmt.Print(formattedTime, " [", class, "][", method, "][", level.String(), "] - ", red, message, reset, "###############\n")
			} else if level <= Fatal {
				fmt.Println("******************************************************************************************************************")
				fmt.Print(formattedTime, " [", class, "][", method, "][", level.String(), "] - ", red, message, reset, "\n")
				fmt.Println("******************************************************************************************************************")
			} else {
				fmt.Print(formattedTime, " [", class, "][", method, "][", level.String(), "] - ", message, "\n")
			}
		}
	}
}

func (u *Utils) PrintBanner(product string, version string, copyrightYear string, hashes int) {
	versionString := "Version " + version
	copyrightString := "Copyright " + copyrightYear + "by  James Williams"
	interior := max(len(product), len(versionString), len(copyrightString)) + 6

	headerHashString := strings.Repeat("#", hashes+interior+hashes)
	hashString := strings.Repeat("#", hashes)

	fmt.Println(headerHashString)
	fmt.Println(hashString + u.centerInString(product, interior) + hashString)
	fmt.Println(hashString + u.centerInString("Version "+version, interior) + hashString)
	fmt.Println(hashString + u.centerInString("Copyright "+copyrightYear+" James Williams", interior) + hashString)
	fmt.Println(headerHashString)
	fmt.Println("")
	fmt.Println("")
}

func (u Utils) centerInString(word string, spaces int) string {
	spacesNeeded := spaces - len(word)
	spacesNeededBefore := spacesNeeded / 2
	spacesNeededAfter := spacesNeeded - spacesNeededBefore

	return strings.Repeat(" ", spacesNeededBefore) + word + strings.Repeat(" ", spacesNeededAfter)
}
