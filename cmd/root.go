package cmd

import (
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/K4rian/kfdsl/internal/arguments"
	"github.com/K4rian/kfdsl/internal/settings"
)

func BuildRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "./kfdsl",
		Short: "KF Dedicated Server Launcher",
		Long:  "A command-line tool to configure and run a Killing Floor Dedicated Server.",
		RunE:  runRootCommand,
	}

	var userHome, _ = os.UserHomeDir()

	var configFile, modsFile, serverName, shortName, gameMode, startupMap, gameDifficulty,
		gameLength, password, adminName, adminMail, adminPassword, motd, specimenType, mutators,
		serverMutators, redirectURL, mapList, allTradersMessage, logLevel, logFilePath,
		logFileFormat, steamRootDir, steamAppInstallDir string

	var gamePort, webadminPort, gamespyPort, maxPlayers, maxSpectators, region,
		mapVoteRepeatLimit, logMaxSize, logMaxBackups, logMaxAge int

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
		"steamcmd-root":          {&steamRootDir, "SteamCMD root directory", path.Join(userHome, "steamcmd")},
		"steamcmd-appinstalldir": {&steamAppInstallDir, "server installatation directory", path.Join(userHome, "gameserver")},
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

func runRootCommand(cmd *cobra.Command, args []string) error {
	sett := settings.Get()

	registerArguments(sett)

	if err := sett.Parse(); err != nil {
		return err
	}

	viper.SetDefault("KF_EXTRAARGS", args)
	sett.ExtraArgs = viper.GetStringSlice("KF_EXTRAARGS")
	return nil
}

func registerArguments(sett *settings.KFDSLSettings) {
	sett.ConfigFile = arguments.NewArgument("Config File", viper.GetString("config"), nil, nil, false)
	sett.ModsFile = arguments.NewArgument("Mods File", viper.GetString("mods"), nil, nil, false)
	sett.ServerName = arguments.NewArgument("Server Name", viper.GetString("servername"), arguments.ParseNonEmptyStr, nil, false)
	sett.ShortName = arguments.NewArgument("Short Name", viper.GetString("shortname"), arguments.ParseNonEmptyStr, nil, false)
	sett.GamePort = arguments.NewArgument("Game Port", viper.GetInt("port"), arguments.ParsePort, nil, false)
	sett.WebAdminPort = arguments.NewArgument("WebAdmin Port", viper.GetInt("webadminport"), arguments.ParsePort, nil, false)
	sett.GameSpyPort = arguments.NewArgument("GameSpy Port", viper.GetInt("gamespyport"), arguments.ParsePort, nil, false)
	sett.GameMode = arguments.NewArgument("Game Mode", viper.GetString("gamemode"), arguments.ParseGameMode, arguments.FormatGameMode, false)
	sett.StartupMap = arguments.NewArgument("Startup Map", viper.GetString("map"), arguments.ParseNonEmptyStr, nil, false)
	sett.GameDifficulty = arguments.NewArgument("Game Difficulty", settings.DefaultInternalGameDifficulty, arguments.ParseGameDifficulty(viper.GetString("difficulty")), arguments.FormatGameDifficulty, false)
	sett.GameLength = arguments.NewArgument("Game Length", settings.DefaultInternalGameLength, arguments.ParseGameLength(viper.GetString("length")), arguments.FormatGameLength, false)
	sett.FriendlyFire = arguments.NewArgument("Friendly Fire Rate", viper.GetFloat64("friendlyfire"), arguments.ParseFriendlyFireRate, arguments.FormatFriendlyFireRate, false)
	sett.MaxPlayers = arguments.NewArgument("Max Players", viper.GetInt("maxplayers"), nil, nil, false)
	sett.MaxSpectators = arguments.NewArgument("Max Spectators", viper.GetInt("maxspectators"), nil, nil, false)
	sett.Password = arguments.NewArgument("Game Password", viper.GetString("password"), arguments.ParsePassword, nil, true)
	sett.Region = arguments.NewArgument("Region", viper.GetInt("region"), arguments.ParseUnsignedInt, nil, false)
	sett.AdminName = arguments.NewArgument("Admin Name", viper.GetString("adminname"), nil, nil, false)
	sett.AdminMail = arguments.NewArgument("Admin Mail", viper.GetString("adminmail"), arguments.ParseMail, nil, true)
	sett.AdminPassword = arguments.NewArgument("Admin Password", viper.GetString("adminpassword"), arguments.ParsePassword, nil, true)
	sett.MOTD = arguments.NewArgument("MOTD", viper.GetString("motd"), nil, nil, false)
	sett.SpecimenType = arguments.NewArgument("Specimens Type", viper.GetString("specimentype"), arguments.ParseSpecimenType, arguments.FormatSpecimenType, false)
	sett.Mutators = arguments.NewArgument("Mutators", viper.GetString("mutators"), nil, nil, false)
	sett.ServerMutators = arguments.NewArgument("Server Mutators", viper.GetString("servermutators"), nil, nil, false)
	sett.RedirectURL = arguments.NewArgument("Redirect URL", viper.GetString("redirecturl"), arguments.ParseURL, nil, false)
	sett.Maplist = arguments.NewArgument("Maplist", viper.GetString("maplist"), nil, nil, false)
	sett.EnableWebAdmin = arguments.NewArgument("Web Admin", viper.GetBool("webadmin"), nil, arguments.FormatBool, false)
	sett.EnableMapVote = arguments.NewArgument("Map Voting", viper.GetBool("mapvote"), nil, arguments.FormatBool, false)
	sett.MapVoteRepeatLimit = arguments.NewArgument("Map Vote Repeat Limit", viper.GetInt("mapvote-repeatlimit"), arguments.ParseUnsignedInt, nil, false)
	sett.EnableAdminPause = arguments.NewArgument("Admin Pause", viper.GetBool("adminpause"), nil, arguments.FormatBool, false)
	sett.DisableWeaponThrow = arguments.NewArgument("No Weapon Throw", viper.GetBool("noweaponthrow"), nil, arguments.FormatBool, false)
	sett.DisableWeaponShake = arguments.NewArgument("No Weapon Shake", viper.GetBool("noweaponshake"), nil, arguments.FormatBool, false)
	sett.EnableThirdPerson = arguments.NewArgument("Third Person View", viper.GetBool("thirdperson"), nil, arguments.FormatBool, false)
	sett.EnableLowGore = arguments.NewArgument("Low Gore", viper.GetBool("lowgore"), nil, arguments.FormatBool, false)
	sett.Uncap = arguments.NewArgument("Uncap Framerate", viper.GetBool("uncap"), nil, arguments.FormatBool, false)
	sett.Unsecure = arguments.NewArgument("Unsecure (no VAC)", viper.GetBool("unsecure"), nil, arguments.FormatBool, false)
	sett.NoSteam = arguments.NewArgument("Skip SteamCMD", viper.GetBool("nosteam"), nil, arguments.FormatBool, false)
	sett.NoValidate = arguments.NewArgument("Files Validation", viper.GetBool("novalidate"), nil, arguments.FormatBool, false)
	sett.AutoRestart = arguments.NewArgument("Server Auto Restart", viper.GetBool("autorestart"), nil, arguments.FormatBool, false)
	sett.EnableMutLoader = arguments.NewArgument("Use MutLoader", viper.GetBool("mutloader"), nil, arguments.FormatBool, false)
	sett.EnableKFPatcher = arguments.NewArgument("Use KFPatcher", viper.GetBool("kfpatcher"), nil, arguments.FormatBool, false)
	sett.KFPHidePerks = arguments.NewArgument("KFP Hide Perks", viper.GetBool("hideperks"), nil, arguments.FormatBool, false)
	sett.KFPDisableZedTime = arguments.NewArgument("KFP Disable ZED Time", viper.GetBool("nozedtime"), nil, arguments.FormatBool, false)
	sett.KFPBuyEverywhere = arguments.NewArgument("KFP Buy Everywhere", viper.GetBool("buyeverywhere"), nil, arguments.FormatBool, false)
	sett.KFPEnableAllTraders = arguments.NewArgument("KFP All Traders", viper.GetBool("alltraders"), nil, arguments.FormatBool, false)
	sett.KFPAllTradersMessage = arguments.NewArgument("KFP All Traders Msg", viper.GetString("alltraders-message"), nil, nil, false)
	sett.LogToFile = arguments.NewArgument("Log to File", viper.GetBool("log-to-file"), nil, arguments.FormatBool, false)
	sett.LogLevel = arguments.NewArgument("Log Level", viper.GetString("log-level"), arguments.ParseLogLevel, nil, false)
	sett.LogFile = arguments.NewArgument("Log File", viper.GetString("log-file"), nil, nil, false)
	sett.LogFileFormat = arguments.NewArgument("Log File Format", viper.GetString("log-file-format"), arguments.ParseLogFileFormat, nil, false)
	sett.LogMaxSize = arguments.NewArgument("Log Max Size (MB)", viper.GetInt("log-max-size"), arguments.ParsePositiveInt, nil, false)
	sett.LogMaxBackups = arguments.NewArgument("Log Max Backups", viper.GetInt("log-max-backups"), arguments.ParsePositiveInt, nil, false)
	sett.LogMaxAge = arguments.NewArgument("Log Max Age (days)", viper.GetInt("log-max-age"), arguments.ParsePositiveInt, nil, false)

	sett.MaxPlayers.SetParserFunction(arguments.ParseIntRange(sett.MaxPlayers, 0, 32))
	sett.MaxSpectators.SetParserFunction(arguments.ParseIntRange(sett.MaxSpectators, 0, 32))
}
