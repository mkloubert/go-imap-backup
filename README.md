# go-imap-backup

> Tool that back ups IMAP messages to local .eml files, written in Go.

## Usage

Change to [project root](./) and first create an [.env file](./.env) like this:

```dotenv
# 1st example configuration for GMX
IMAP_BACKUP_1=example@gmx.de
IMAP_BACKUP_1_IMAP_HOST=imap.gmx.net
IMAP_BACKUP_1_IMAP_PORT=993
IMAP_BACKUP_1_IMAP_USER=example@gmx.de
IMAP_BACKUP_1_IMAP_PASSWORD=<YOUR-GMX-APP-SPECIFIC-PASSWORD>

# 2nd example configuration for GMAIL
IMAP_BACKUP_2=example@gmail.com
IMAP_BACKUP_2_IMAP_HOST=imap.gmail.com
IMAP_BACKUP_2_IMAP_PORT=993
IMAP_BACKUP_2_IMAP_USER=example@gmail.com
IMAP_BACKUP_2_IMAP_PASSWORD=<YOUR-GMAIL-APP-SPECIFIC-PASSWORD>

# 3rd example configuration for iCloud
IMAP_BACKUP_3=example@icloud.com
IMAP_BACKUP_3_IMAP_HOST=imap.mail.me.com
IMAP_BACKUP_3_IMAP_PORT=993
IMAP_BACKUP_3_IMAP_USER=example
IMAP_BACKUP_3_IMAP_PASSWORD=<YOUR-ICLOUD-APP-SPECIFIC-PASSWORD>
```

and finally run

```bash
go run .
```
