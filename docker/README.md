# KF Dedicated Server Launcher (KFDSL)

A Docker image for the [Killing Floor Dedicated Server Launcher (kfdsl)][1] based on a [Debian][2] [image][3].

---

## Environment variables
Several environment variables can be tweaked when creating a container to define the server configuration:

<details>
<summary>[Click to expand]</summary>

Variable               | Default value                 | Description
---                    | ---                           | ---
KF_ADMINMAIL           |                               | server administrator email
KF_ADMINNAME           |                               | server administrator name
KF_ADMINPASSWORD       |                               | server administrator password
KF_ADMINPAUSE          | false                         | allow admin to pause game
KF_ALLTRADERS          | false                         | (KFPatcher) make all trader's spots accessible
KF_ALLTRADERS-MESSAGE  | "^wAll traders are ^ropen^w!" | (KFPatcher) All traders screen message
KF_AUTORESTART         | false                         | restart server on crash
KF_BUYEVERYWHERE       | false                         | (KFPatcher) allow players to shop whenever
KF_CONFIG              | KillingFloor.ini              | configuration file
KF_DIFFICULTY          | hard                          | game difficulty (easy, normal, hard, suicidal, hell)
KF_FRIENDLYFIRE        | 0                             | friendly fire rate (0.0-1.0)
KF_GAMEMODE            | survival                      | game mode
KF_GAMESPYPORT         | 7717                          | GameSpy UDP port
KF_HIDEPERKS           | false                         | (KFPatcher) hide perks
KF_KFPATCHER           | false                         | enable KFPatcher
KF_LENGTH              | medium                        | game length (waves) (short, medium, long)
KF_LOG-FILE            | ./kfdsl.log                   | log file path
KF_LOG-FILE-FORMAT     | text                          | log format (text or json)
KF_LOG-LEVEL           | info                          | log level (info, debug, warn, error)
KF_LOG-MAX-AGE         | 28                            | max age of a log file (days)
KF_LOG-MAX-BACKUPS     | 5                             | max number of old log files to keep
KF_LOG-MAX-SIZE        | 10                            | max log file size (MB)
KF_LOG-TO-FILE         | false                         | enable file logging
KF_LOWGORE             | false                         | reduce gore
KF_MAP                 | KF-BioticsLab                 | starting map
KF_MAPLIST             | all                           | comma-separated maps for the current game mode. Use 'all' to append all available map
KF_MAPVOTE             | false                         | enable map voting
KF_MAPVOTE-REPEATLIMIT | 1                             | number of maps to be played before a map can repeat
KF_MAXPLAYERS          | 6                             | maximum players
KF_MAXSPECTATORS       | 6                             | maximum spectators
KF_MOTD                |                               | message of the day
KF_MUTATORS            |                               | comma-separated mutators (command-line)
KF_MUTLOADER           | false                         | enable MutLoader (override inline mutators)
KF_NOSTEAM             | false                         | start the server without calling SteamCMD
KF_NOVALIDATE          | false                         | skip server files integrity check
KF_NOWEAPONSHAKE       | false                         | disable weapon-induced screen shake
KF_NOWEAPONTHROW       | false                         | disable weapon throwing
KF_NOZEDTIME           | false                         | (KFPatcher) disable ZED time
KF_PASSWORD            |                               | server password
KF_PORT                | 7707                          | game UDP port
KF_REDIRECTURL         |                               | redirect URL
KF_REGION              | 1                             | server region
KF_SERVERMUTATORS      |                               | comma-separated mutators (server actors)
KF_SERVERNAME          | KF Server                     | server name
KF_SHORTNAME           | KFS                           | server short name
KF_SPECIMENTYPE        | default                       | specimen type (default, summer, halloween, christmas)
KF_THIRDPERSON         | false                         | enable third-person view
KF_UNCAP               | false                         | uncap the frame rate
KF_UNSECURE            | false                         | disable VAC (Valve Anti-Cheat)
KF_WEBADMIN            | false                         | enable WebAdmin panel
KF_WEBADMINPORT        | 8075                          | WebAdmin TCP port
STEAMACC_PASSWORD      |                               | password of the steam account
STEAMACC_USERNAME      | anonymous                     | username of the steam account
STEAMCMD_ROOT          | ~/steamcmd                    | directory where steamcmd will be stored
STEAMCMD_APPINSTALLDIR | ~/gameserver                  | directory where the gameserver files will be stored

</details>

## Usage
Run the server using default configuration.<br>
For the Killing Floor 1 gameserver to be downloaded, make sure to set `STEAMACC_USERNAME` and `STEAMACC_PASSWORD` environment variables to match a valid Steam account.

```bash
docker run -d \
  --name kfdsl \
  -p 7707:7707/udp \
  -p 7708:7708/udp \
  -p 8075:8075/tcp \
  -p 20560:20560/udp \
  -p 28852:28852/tcp \
  -p 28852:28852/udp \
  -i k4rian/kfdsl
```

## Using Compose
See the [docker-compose.yml][4] file.

## Manual build
__Requirements__:<br>
— Docker >= __18.09.0__<br>
— Git *(optional)*

Like any Docker image the building process is pretty straightforward: 

- Clone (or download) the GitHub repository to an empty folder on your local machine:
```bash
git clone https://github.com/K4rian/kfdsl.git .
```

- Then run the following command inside the newly created folder:
```bash
cd docker/ && docker build --no-cache -t k4rian/kfdsl .
```

[1]: https://github.com/K4rian/kfdsl
[2]: https://www.debian.org/ "Debian Official Website"
[3]: https://github.com/K4rian/docker-steamcmd "steamcmd Docker Image"
[4]: https://github.com/K4rian/kfdsl/blob/main/docker/docker-compose.yml
