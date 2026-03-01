package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/K4rian/kfdsl/internal/arguments"
	"github.com/K4rian/kfdsl/internal/settings"
)

func BuildRootCommand(sett *settings.Settings) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "./kfdsl",
		Short: "KF Dedicated Server Launcher",
		Long:  "A command-line tool to configure and run a Killing Floor Dedicated Server.",
		RunE: func(cmd *cobra.Command, args []string) error {
			registerArguments(sett)

			if err := sett.Parse(); err != nil {
				return err
			}
			viper.SetDefault("KF_EXTRAARGS", args)
			sett.ExtraArgs = viper.GetStringSlice("KF_EXTRAARGS")
			return nil
		},
	}

	var userHome, _ = os.UserHomeDir()

	var configFile, modsFile, serverName, shortName, gameMode, startupMap, gameDifficulty,
		gameLength, password, adminName, adminMail, adminPassword, motd, specimenType, mutators,
		serverMutators, redirectURL, mapList, allTradersMessage, logLevel, logFilePath,
		logFileFormat, steamRootDir, steamAppInstallDir string

	var gamePort, webadminPort, gamespyPort, maxPlayers, maxSpectators, region,
		mapVoteRepeatLimit, logMaxSize, logMaxBackups, logMaxAge,
		maxRestarts, restartDelay, shutdownTimeout, killTimeout int

	var friendlyFire float64

	var enableWebAdmin, enableMapVote, enableAdminPause, disableWeaponThrow,
		disableWeaponShake, enableThirdPerson, enableLowGore, uncap, unsecure, noSteam,
		disableValidation, enableAutoRestart, enableMutloader, enableKFPatcher, enableShowPerks,
		disableZEDTime, enableBuyEverywhere, enableAllTraders, enableFileLogging bool

	flags := map[string]struct {
		Value   interface{}
		Desc    string
		Default interface{}
	}{
		"mods":                   {&modsFile, "mods file", settings.DefaultModsFile},
		"config":                 {&configFile, "configuration file", settings.DefaultConfigFile},
		"servername":             {&serverName, "server name", settings.DefaultServerName},
		"shortname":              {&shortName, "server short name", settings.DefaultShortName},
		"port":                   {&gamePort, "game UDP port", settings.DefaultGamePort},
		"webadminport":           {&webadminPort, "WebAdmin TCP port", settings.DefaultWebAdminPort},
		"gamespyport":            {&gamespyPort, "GameSpy UDP port", settings.DefaultGameSpyPort},
		"gamemode":               {&gameMode, "game mode", settings.DefaultGameMode},
		"map":                    {&startupMap, "starting map", settings.DefaultStartupMap},
		"difficulty":             {&gameDifficulty, "game difficulty (easy, normal, hard, suicidal, hell)", settings.DefaultGameDifficulty},
		"length":                 {&gameLength, "game length (waves) (short, medium, long)", settings.DefaultGameLength},
		"friendlyfire":           {&friendlyFire, "friendly fire rate (0.0-1.0)", settings.DefaultFriendlyFire},
		"maxplayers":             {&maxPlayers, "maximum players", settings.DefaultMaxPlayers},
		"maxspectators":          {&maxSpectators, "maximum spectators", settings.DefaultMaxSpectators},
		"password":               {&password, "server password", settings.DefaultPassword},
		"region":                 {&region, "server region", settings.DefaultRegion},
		"adminname":              {&adminName, "server administrator name", settings.DefaultAdminName},
		"adminmail":              {&adminMail, "server administrator email", settings.DefaultAdminMail},
		"adminpassword":          {&adminPassword, "server administrator password", settings.DefaultAdminPassword},
		"motd":                   {&motd, "message of the day", settings.DefaultMOTD},
		"specimentype":           {&specimenType, "specimen type (default, summer, halloween, christmas)", settings.DefaultSpecimenType},
		"mutators":               {&mutators, "comma-separated mutators (command-line)", settings.DefaultMutators},
		"servermutators":         {&serverMutators, "comma-separated mutators (server actors)", settings.DefaultServerMutators},
		"redirecturl":            {&redirectURL, "redirect URL", settings.DefaultRedirectURL},
		"maplist":                {&mapList, "comma-separated maps for the current game mode. Use 'all' to append all available map", settings.DefaultMaplist},
		"webadmin":               {&enableWebAdmin, "enable WebAdmin panel", settings.DefaultEnableWebAdmin},
		"mapvote":                {&enableMapVote, "enable map voting", settings.DefaultEnableMapVote},
		"mapvote-repeatlimit":    {&mapVoteRepeatLimit, "number of maps to be played before a map can repeat", settings.DefaultMapVoteRepeatLimit},
		"adminpause":             {&enableAdminPause, "allow admin to pause game", settings.DefaultEnableAdminPause},
		"noweaponthrow":          {&disableWeaponThrow, "disable weapon throwing", settings.DefaultDisableWeaponThrow},
		"noweaponshake":          {&disableWeaponShake, "disable weapon-induced screen shake", settings.DefaultDisableWeaponShake},
		"thirdperson":            {&enableThirdPerson, "enable third-person view", settings.DefaultEnableThirdPerson},
		"lowgore":                {&enableLowGore, "reduce gore", settings.DefaultEnableLowGore},
		"uncap":                  {&uncap, "uncap the frame rate", settings.DefaultUncap},
		"unsecure":               {&unsecure, "disable VAC (Valve Anti-Cheat)", settings.DefaultUnsecure},
		"nosteam":                {&noSteam, "start the server without calling SteamCMD", settings.DefaultNoSteam},
		"novalidate":             {&disableValidation, "skip server files integrity check", settings.DefaultNoValidate},
		"autorestart":            {&enableAutoRestart, "restart server on crash", settings.DefaultAutoRestart},
		"mutloader":              {&enableMutloader, "enable MutLoader (override inline mutators)", settings.DefaultEnableMutLoader},
		"kfpatcher":              {&enableKFPatcher, "enable KFPatcher", settings.DefaultEnableKFPatcher},
		"hideperks":              {&enableShowPerks, "(KFPatcher) hide perks", settings.DefaultKFPHidePerks},
		"nozedtime":              {&disableZEDTime, "(KFPatcher) disable ZED time", settings.DefaultKFPDisableZedTime},
		"buyeverywhere":          {&enableBuyEverywhere, "(KFPatcher) allow players to shop whenever", settings.DefaultKFPBuyEverywhere},
		"alltraders":             {&enableAllTraders, "(KFPatcher) make all trader's spots accessible", settings.DefaultKFPEnableAllTraders},
		"alltraders-message":     {&allTradersMessage, "(KFPatcher) All traders screen message", settings.DefaultKFPAllTradersMessage},
		"log-to-file":            {&enableFileLogging, "enable file logging", settings.DefaultLogToFile},
		"log-level":              {&logLevel, "log level (info, debug, warn, error)", settings.DefaultLogLevel},
		"log-file":               {&logFilePath, "log file path", settings.DefaultLogFile},
		"log-file-format":        {&logFileFormat, "log format (text or json)", settings.DefaultLogFileFormat},
		"log-max-size":           {&logMaxSize, "max log file size (MB)", settings.DefaultLogMaxSize},
		"log-max-backups":        {&logMaxBackups, "max number of old log files to keep", settings.DefaultLogMaxBackups},
		"log-max-age":            {&logMaxAge, "max age of a log file (days)", settings.DefaultLogMaxAge},
		"max-restarts":           {&maxRestarts, "max restart attempts in a row", settings.DefaultMaxRestarts},
		"restart-delay":          {&restartDelay, "delay between restart (in secs)", settings.DefaultRestartDelay},
		"shutdown-timeout":       {&shutdownTimeout, "server shutdown timeout (in secs)", settings.DefaultShutdownTimeout},
		"kill-timeout":           {&killTimeout, "server process kill timeout (in secs)", settings.DefaultKillTimeout},
		"steamcmd-root":          {&steamRootDir, "SteamCMD root directory", filepath.Join(userHome, "steamcmd")},
		"steamcmd-appinstalldir": {&steamAppInstallDir, "server installatation directory", filepath.Join(userHome, "gameserver")},
	}

	for flag, data := range flags {
		switch v := data.Default.(type) {
		case string:
			val := data.Value.(*string)
			rootCmd.Flags().StringVar(val, flag, v, data.Desc)
		case int:
			val := data.Value.(*int)
			rootCmd.Flags().IntVar(val, flag, v, data.Desc)
		case float64:
			val := data.Value.(*float64)
			rootCmd.Flags().Float64Var(val, flag, v, data.Desc)
		case bool:
			val := data.Value.(*bool)
			rootCmd.Flags().BoolVar(val, flag, v, data.Desc)
		}

		// SteamCMD-related configurations don't use the 'KF' prefix
		if strings.HasPrefix(flag, "steamcmd") {
			envName := strings.ToUpper(strings.ReplaceAll(flag, "-", "_"))
			viper.BindEnv(flag, envName)
		} else {
			viper.BindEnv(flag)
		}
		viper.BindPFlag(flag, rootCmd.Flags().Lookup(flag))
	}

	viper.BindEnv("STEAMACC_USERNAME")
	viper.BindEnv("STEAMACC_PASSWORD")
	viper.BindEnv("KF_EXTRAARGS")

	viper.SetDefault("STEAMACC_USERNAME", settings.DefaultSteamLogin)

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("KF")
	viper.AutomaticEnv()

	return rootCmd
}

func registerArguments(sett *settings.Settings) {
	sett.ConfigFile = arguments.New("Config File", viper.GetString("config"), nil, nil, false)
	sett.ModsFile = arguments.New("Mods File", viper.GetString("mods"), nil, nil, false)
	sett.ServerName = arguments.New("Server Name", viper.GetString("servername"), arguments.ParseNonEmptyStr, nil, false)
	sett.ShortName = arguments.New("Short Name", viper.GetString("shortname"), arguments.ParseNonEmptyStr, nil, false)
	sett.GamePort = arguments.New("Game Port", viper.GetInt("port"), arguments.ParsePort, nil, false)
	sett.WebAdminPort = arguments.New("WebAdmin Port", viper.GetInt("webadminport"), arguments.ParsePort, nil, false)
	sett.GameSpyPort = arguments.New("GameSpy Port", viper.GetInt("gamespyport"), arguments.ParsePort, nil, false)
	sett.GameMode = arguments.New("Game Mode", viper.GetString("gamemode"), arguments.ParseGameMode, arguments.FormatGameMode, false)
	sett.StartupMap = arguments.New("Startup Map", viper.GetString("map"), arguments.ParseNonEmptyStr, nil, false)
	sett.GameDifficulty = arguments.New("Game Difficulty", settings.DefaultInternalGameDifficulty, arguments.ParseGameDifficulty(viper.GetString("difficulty")), arguments.FormatGameDifficulty, false)
	sett.GameLength = arguments.New("Game Length", settings.DefaultInternalGameLength, arguments.ParseGameLength(viper.GetString("length")), arguments.FormatGameLength, false)
	sett.FriendlyFire = arguments.New("Friendly Fire Rate", viper.GetFloat64("friendlyfire"), arguments.ParseFriendlyFireRate, arguments.FormatFriendlyFireRate, false)
	sett.MaxPlayers = arguments.New("Max Players", viper.GetInt("maxplayers"), nil, nil, false)
	sett.MaxSpectators = arguments.New("Max Spectators", viper.GetInt("maxspectators"), nil, nil, false)
	sett.Password = arguments.New("Game Password", viper.GetString("password"), arguments.ParsePassword, nil, true)
	sett.Region = arguments.New("Region", viper.GetInt("region"), arguments.ParseUnsignedInt, nil, false)
	sett.AdminName = arguments.New("Admin Name", viper.GetString("adminname"), nil, nil, false)
	sett.AdminMail = arguments.New("Admin Mail", viper.GetString("adminmail"), arguments.ParseMail, nil, true)
	sett.AdminPassword = arguments.New("Admin Password", viper.GetString("adminpassword"), arguments.ParsePassword, nil, true)
	sett.MOTD = arguments.New("MOTD", viper.GetString("motd"), nil, nil, false)
	sett.SpecimenType = arguments.New("Specimens Type", viper.GetString("specimentype"), arguments.ParseSpecimenType, arguments.FormatSpecimenType, false)
	sett.Mutators = arguments.New("Mutators", viper.GetString("mutators"), nil, nil, false)
	sett.ServerMutators = arguments.New("Server Mutators", viper.GetString("servermutators"), nil, nil, false)
	sett.RedirectURL = arguments.New("Redirect URL", viper.GetString("redirecturl"), arguments.ParseURL, nil, false)
	sett.Maplist = arguments.New("Maplist", viper.GetString("maplist"), nil, nil, false)
	sett.EnableWebAdmin = arguments.New("Web Admin", viper.GetBool("webadmin"), nil, arguments.FormatBool, false)
	sett.EnableMapVote = arguments.New("Map Voting", viper.GetBool("mapvote"), nil, arguments.FormatBool, false)
	sett.MapVoteRepeatLimit = arguments.New("Map Vote Repeat Limit", viper.GetInt("mapvote-repeatlimit"), arguments.ParseUnsignedInt, nil, false)
	sett.EnableAdminPause = arguments.New("Admin Pause", viper.GetBool("adminpause"), nil, arguments.FormatBool, false)
	sett.DisableWeaponThrow = arguments.New("No Weapon Throw", viper.GetBool("noweaponthrow"), nil, arguments.FormatBool, false)
	sett.DisableWeaponShake = arguments.New("No Weapon Shake", viper.GetBool("noweaponshake"), nil, arguments.FormatBool, false)
	sett.EnableThirdPerson = arguments.New("Third Person View", viper.GetBool("thirdperson"), nil, arguments.FormatBool, false)
	sett.EnableLowGore = arguments.New("Low Gore", viper.GetBool("lowgore"), nil, arguments.FormatBool, false)
	sett.Uncap = arguments.New("Uncap Framerate", viper.GetBool("uncap"), nil, arguments.FormatBool, false)
	sett.Unsecure = arguments.New("Unsecure (no VAC)", viper.GetBool("unsecure"), nil, arguments.FormatBool, false)
	sett.NoSteam = arguments.New("Skip SteamCMD", viper.GetBool("nosteam"), nil, arguments.FormatBool, false)
	sett.NoValidate = arguments.New("Files Validation", viper.GetBool("novalidate"), nil, arguments.FormatBool, false)
	sett.AutoRestart = arguments.New("Server Auto Restart", viper.GetBool("autorestart"), nil, arguments.FormatBool, false)
	sett.EnableMutLoader = arguments.New("Use MutLoader", viper.GetBool("mutloader"), nil, arguments.FormatBool, false)
	sett.EnableKFPatcher = arguments.New("Use KFPatcher", viper.GetBool("kfpatcher"), nil, arguments.FormatBool, false)
	sett.KFPHidePerks = arguments.New("KFP Hide Perks", viper.GetBool("hideperks"), nil, arguments.FormatBool, false)
	sett.KFPDisableZedTime = arguments.New("KFP Disable ZED Time", viper.GetBool("nozedtime"), nil, arguments.FormatBool, false)
	sett.KFPBuyEverywhere = arguments.New("KFP Buy Everywhere", viper.GetBool("buyeverywhere"), nil, arguments.FormatBool, false)
	sett.KFPEnableAllTraders = arguments.New("KFP All Traders", viper.GetBool("alltraders"), nil, arguments.FormatBool, false)
	sett.KFPAllTradersMessage = arguments.New("KFP All Traders Msg", viper.GetString("alltraders-message"), nil, nil, false)
	sett.LogToFile = arguments.New("Log to File", viper.GetBool("log-to-file"), nil, arguments.FormatBool, false)
	sett.LogLevel = arguments.New("Log Level", viper.GetString("log-level"), arguments.ParseLogLevel, nil, false)
	sett.LogFile = arguments.New("Log File", viper.GetString("log-file"), nil, nil, false)
	sett.LogFileFormat = arguments.New("Log File Format", viper.GetString("log-file-format"), arguments.ParseLogFileFormat, nil, false)
	sett.LogMaxSize = arguments.New("Log Max Size (MB)", viper.GetInt("log-max-size"), arguments.ParsePositiveInt, nil, false)
	sett.LogMaxBackups = arguments.New("Log Max Backups", viper.GetInt("log-max-backups"), arguments.ParsePositiveInt, nil, false)
	sett.LogMaxAge = arguments.New("Log Max Age (days)", viper.GetInt("log-max-age"), arguments.ParsePositiveInt, nil, false)
	sett.MaxRestarts = arguments.New("Max Restarts", viper.GetInt("max-restarts"), arguments.ParseUnsignedInt, nil, false)
	sett.RestartDelay = arguments.New("Restart Delay (secs)", viper.GetDuration("restart-delay"), arguments.ParseDuration, nil, false)
	sett.ShutdownTimeout = arguments.New("Shutdown Timeout (secs)", viper.GetDuration("shutdown-timeout"), arguments.ParseDuration, nil, false)
	sett.KillTimeout = arguments.New("Kill Timeout (secs)", viper.GetDuration("kill-timeout"), arguments.ParseDuration, nil, false)
	sett.SteamCMDRoot = arguments.New("SteamCMD Root", viper.GetString("steamcmd-root"), arguments.ParseExistingDir, nil, false)
	sett.ServerInstallDir = arguments.New("Server Install Dir", viper.GetString("steamcmd-appinstalldir"), arguments.ParseExistingDir, nil, false)

	sett.MaxPlayers.SetParserFunction(arguments.ParseIntRange(sett.MaxPlayers, 0, 32))
	sett.MaxSpectators.SetParserFunction(arguments.ParseIntRange(sett.MaxSpectators, 0, 32))
}
