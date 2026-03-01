package settings

import (
	"fmt"
	"reflect"
	"time"

	"github.com/K4rian/kfdsl/internal/arguments"
	"github.com/K4rian/kfdsl/internal/log"
)

type Settings struct {
	ConfigFile           *arguments.Argument[string]        // Server Configuration File
	ModsFile             *arguments.Argument[string]        // File defining which mods to install
	ServerName           *arguments.Argument[string]        // Server Name
	ShortName            *arguments.Argument[string]        // Server Alias
	GamePort             *arguments.Argument[int]           // Port
	WebAdminPort         *arguments.Argument[int]           // Web Admin Panel Port
	GameSpyPort          *arguments.Argument[int]           // GameSpy Port
	GameMode             *arguments.Argument[string]        // Game Mode to use (Survival, Objective, Toy Master or Custom)
	StartupMap           *arguments.Argument[string]        // Starting map
	GameDifficulty       *arguments.Argument[int]           // Game Difficulty
	GameLength           *arguments.Argument[int]           // Game Length
	FriendlyFire         *arguments.Argument[float64]       // Friendly Fire Rate
	MaxPlayers           *arguments.Argument[int]           // Maximum Players
	MaxSpectators        *arguments.Argument[int]           // Maximum Spectators
	Password             *arguments.Argument[string]        // Server Password
	Region               *arguments.Argument[int]           // Server Region
	AdminName            *arguments.Argument[string]        // Administrator Name
	AdminMail            *arguments.Argument[string]        // Administrator Email address
	AdminPassword        *arguments.Argument[string]        // Administrator Password
	MOTD                 *arguments.Argument[string]        // Message of the Day
	SpecimenType         *arguments.Argument[string]        // Specimen type to use
	Mutators             *arguments.Argument[string]        // Mutators list (Command-line)
	ServerMutators       *arguments.Argument[string]        // Mutators list (ServerActors)
	RedirectURL          *arguments.Argument[string]        // Redirection URL (extra content)
	Maplist              *arguments.Argument[string]        // Map list
	EnableWebAdmin       *arguments.Argument[bool]          // Enable the Web Admin Panel
	EnableMapVote        *arguments.Argument[bool]          // Enable Map voting
	MapVoteRepeatLimit   *arguments.Argument[int]           // Map vote repeat limit (number of maps to be played before a map can repeat)
	EnableAdminPause     *arguments.Argument[bool]          // Allow the administrator(s) to pause the game
	DisableWeaponThrow   *arguments.Argument[bool]          // Prevent the weapons from being thrown on the ground
	DisableWeaponShake   *arguments.Argument[bool]          // Prevent the weapons from shaking the screen
	EnableThirdPerson    *arguments.Argument[bool]          // Enable third-person view (using F4)
	EnableLowGore        *arguments.Argument[bool]          // Disable the gore system (specimens can't be dismembered)
	Uncap                *arguments.Argument[bool]          // Uncap the framerate (must also be tweaked in the client)
	Unsecure             *arguments.Argument[bool]          // Start the server without Valve Anti-Cheat (VAC)
	NoSteam              *arguments.Argument[bool]          // Bypass SteamCMD and start the server right away
	NoValidate           *arguments.Argument[bool]          // Skip server files integrity check
	AutoRestart          *arguments.Argument[bool]          // Auto restart the server if it crashes
	EnableMutLoader      *arguments.Argument[bool]          // Enable MutLoader (https://github.com/Bleeding-Action-Man/MutLoader)
	EnableKFPatcher      *arguments.Argument[bool]          // Enable KFPatcher (https://github.com/InsultingPros/KFPatcher)
	KFPHidePerks         *arguments.Argument[bool]          // KFPatcher: Hide Perks
	KFPDisableZedTime    *arguments.Argument[bool]          // KFPatcher: Disable ZED Time
	KFPBuyEverywhere     *arguments.Argument[bool]          // KFPatcher: Allows opening the buy menu anywhere (untested)
	KFPEnableAllTraders  *arguments.Argument[bool]          // KFPatcher: All of the trader's spots are accessible after each wave
	KFPAllTradersMessage *arguments.Argument[string]        // KFPatcher: All traders open message
	LogToFile            *arguments.Argument[bool]          // Enable file logging
	LogLevel             *arguments.Argument[string]        // Log level (info, debug, warn, error)
	LogFile              *arguments.Argument[string]        // Log file path
	LogFileFormat        *arguments.Argument[string]        // Log format (text or json)
	LogMaxSize           *arguments.Argument[int]           // Max log file size (MB)
	LogMaxBackups        *arguments.Argument[int]           // Max number of old log files to keep
	LogMaxAge            *arguments.Argument[int]           // Max age of a log file (days)
	MaxRestarts          *arguments.Argument[int]           // Max restart in a row in case of crash
	RestartDelay         *arguments.Argument[time.Duration] // Delay between restart in seconds
	ShutdownTimeout      *arguments.Argument[time.Duration] // Server shutdown timeout in seconds
	KillTimeout          *arguments.Argument[time.Duration] // Server process kill timeout in seconds
	SteamCMDRoot         *arguments.Argument[string]        // SteamCMD root directory
	ServerInstallDir     *arguments.Argument[string]        // Server install directory
	ExtraArgs            []string                           // Extra arguments passed to the server
	SteamLogin           string                             // Steam Account Login Username
	SteamPassword        string                             // Steam Account Login Password
}

func (s *Settings) Parse() error {
	val := reflect.ValueOf(s).Elem()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			fieldVal := field.Interface()
			if parsable, ok := fieldVal.(arguments.ParsableArgument); ok {
				if err := parsable.Parse(); err != nil {
					return fmt.Errorf("%w", err)
				}
			} else {
				// Shouldn't happen
				return fmt.Errorf("field '%s' cannot be parsed", val.Type().Field(i).Name)
			}
		}
	}
	return nil
}

func (s *Settings) Print() {
	val := reflect.ValueOf(s).Elem()

	getParsableField := func(v reflect.Value) arguments.ParsableArgument {
		if v.Kind() == reflect.Ptr && !v.IsNil() {
			vVal := v.Interface()
			if parsable, ok := vVal.(arguments.ParsableArgument); ok {
				return parsable
			}
		}
		return nil
	}

	maxKeyLength := 0
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if pField := getParsableField(field); pField != nil {
			nameLen := len(pField.Name())
			if nameLen > maxKeyLength {
				maxKeyLength = nameLen
			}
		}
	}

	log.Logger.Info("====================================================")
	log.Logger.Info("                   KFDSL Settings                   ")
	log.Logger.Info("====================================================")
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if pField := getParsableField(field); pField != nil {
			value := pField.FormattedValue()
			if pField.IsSensitive() {
				if value != "" {
					value = "Yes"
				} else {
					value = "No"
				}
			}
			log.Logger.Info(fmt.Sprintf(" ● %-*s → %s", maxKeyLength, pField.Name(), value))
		}
	}
	log.Logger.Info("=====================================================")
}
