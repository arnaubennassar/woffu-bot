# Woffu Bot
Bot that automates the check in / out process of the Woffu web app.
If you set up a telegram bot token you will receive notifications when the bot check in / out or if there is some error.

## Usgae
Use this docker-compose example, editing the environment variables:

```yaml
version: '3.2'

services:
  woffu-bot:
    container_name: woffu-bot
    image: arnaubennassar/woffu-bot:latest
    restart: unless-stopped
    environment:
    - WOFFU_USER=you@your-corp.com
    - WOFFU_PASS=YourStrongPassword
    - CORP=your-corp
    - CHECKIN=9:30
    - CHECKOUT=17:45
    - WORKINGDAYIDS=5
    - TZ=Europe/Madrid
    - IMPRECISSION=300
    - BOT=0987654321:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
    - CHAT=123456789
```

| Name  | Description  |  Example |
|---|---|---|
| WOFFU_USER  | Your Woffu user name  | foo@bar.com  |
| WOFFU_PASS  | Your Woffu password  | TopSecretPass  |
| CORP  | The corporation name that appears in the URL while your logged in (https://e-corp.woffu.com/#/dashboard/user)  | e-corp  |
| CHECKIN  | Time in which you check in  | 9:30  |
| CHECKOUT  | Time in which you check out  | 17:45  |
| WORKINGDAYIDS  | Identifiers of the event types in which you work. List separated by comas. Use the developer tools of your browser to inspect your events to see which identifiers you should check in/out. To be more precise, when logged in check the response of the endpoint `/events?fromDate=...`, in the response look for the `EventTypeId`   | 3,5,7  |
| TZ  | (OPTIONAL) Time Zone. Default is Europe/Madrid  | Europe/Madrid  |
| IMPRECISSION  | (OPTIONAL) Ammount of seconds that randomize the check in/out (check time + rand(IMPRECISSION) - IMPRECISSION/2)  | 300  |
| BOT  | (OPTIONAL) Token of a telegram bot | 1111111111:XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX  |
| CHAT  | (REQUIRED IF BOT IS USED) Telegram chat ID where the bot will send the messages  | 123456789  |

**Note that:** Check in time must be lower than check out time. If you work from 22:00 to 3:00, you should set up the variables like this:

```yaml
# ...
    - CHECKIN=3:00
    - CHECKOUT=22:00
# ...
```