# gae_task_resolver

Task ID をやりとりするだけのGolang製 Web API.
GAE/Flex (Docker)専用。

## API

### GET /get

タスクIDをひとつ取得する。

### POST /finished

タスクの終了を通知する。  
payloadはこちら。

```
{"ID":"FINISHED-TASK-ID"}
```

### POST /add

未完了状態のタスクを登録する。  
payloadはこちら。

```
{"ID":"FINISHED-TASK-ID"}
```

### GET /state

タスクの進捗状態を確認する。

## local test

```
$ go run server.go 
```

## deploy

```
$ docker build -t gcr.io/${your-repository}/task_resolver . # コンテナイメージのビルド
$ docker push gcr.io/${your-repository}/task_resolver # コンテナイメージをGCPにpush
$ gcloud app deploy --project ${your-porject} --image-url  gcr.io/${your-repository}/task_resolver:latest --version ${some-version} . #デプロイ実行
```