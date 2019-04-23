# slack-cmd-bot

## 使用方法

### バイナリの作成

Go 1.12以上の環境を整えた上で

```bash
$ make
```

### SlackのBotトークンを取得

https://api.slack.com/apps にアクセスして"Create a new app"よりアプリケーションを作成。

"Add features and functionality"項の"Bots"ボタンからBotユーザを作成し、同じく"Permissions"ボタンから"Bot User OAuth Access Token"を取得しておく。

また、"Install your app to your workspace"項の"Install App"ボタンを押して自身のSlack Workspaceに追加する。

### config.ymlの記述

* 先に取得したトークンを`Token`項に記述
* Botへのコマンド送信を許可するSlackユーザのEmailアドレスを`Users`項に列挙
* Botが実行できるコマンドを`CmdPattern`項に正規表現で記述

### 実行

PC上でバイナリを起動することでBotを起動する。（実用上はsystemdなどでdaemon化しておくのがおすすめ）

対象となるチャンネルに招待した上で、Botに対してリプライの形でコマンドを投げる。
