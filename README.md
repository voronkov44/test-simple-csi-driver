# CSI-driver
Данный csi-драйвер:
1) Запускается на каждой ноде Kubernetes

2) Создаёт UNIX-сокет (/tmp/simple.csi.driver/csi.sock), через который общается с kubelet

3) При получении запроса на монтирование (когда Pod обращается к PVC):

  - Эмулирует работу CSI, выполняя команду:

    ```bash
    mount -t nullfs /Users/f1lzz/volumes/test-volume /Users/f1lzz/volumes/test-volume
    ```
  - По сути, делает "bind mount" одной и той же директории (ничего не меняет, просто имитирует работу).


Это заглушка для тестирования CSI-интеграции. В реальном драйвере тут было бы реализовано:

- Динамическое создание/удаление томов

- Настоящее монтирование в Pod (например, NFS, iSCSI, облачные диски)

- Обработка ошибок и метаданных
Но для понимания механизма CSI - неплохой учебный пример.

Данный репозиторий был сделан больше для теории и собственного понимания, чтобы разложить все по полочкам.



## Что такое csi и для чего он нужен?
CSI(Container Storage Interface) — это стандартизованный интерфейс (API), который позволяет системам оркестрации контейнеров (например, Kubernetes, Nomad, Mesos) подключаться к любым системам хранения данных (Storage backends) с помощью унифицированного драйвера.

Раньше у каждого оркестратора был свой способ подключения стореджей. Это создаёт сложности для разработчиков стореджей — приходилось делать отдельные интеграции для каждого оркестратора.
С CSI эту проблему решили, стандартизировав способ подключения.

CSI-драйвер работает по архитектуре gRPC и реализует 3 сервиса:
- Identity Service

- Controller Service

- Node Service

Пример команд для каждого сервиса:
- Identity Service: 
  - GetPluginInfo — возвращает имя и версию плагина.

  - GetPluginCapabilities — показывает, что умеет плагин (например, снапшоты или расширение томов).

- Controller Service (работает с томами централизовано):
  - CreateVolume — создать новый том на storage backend.

  - DeleteVolume — удалить том.

  - ControllerPublishVolume — подключить том к ноде (attach).

  - ControllerUnpublishVolume — отключить от ноды.

  - CreateSnapshot — сделать снапшот тома.

  - DeleteSnapshot — удалить снапшот.

- Node Service (подключает уже готовый том на конкретной машине):
  - NodeStageVolume — подготовить том на ноде (например, отформатировать, смонтировать во временную директорию).

  - NodePublishVolume — смонтировать том в нужную директорию для пода.

  - NodeUnpublishVolume — отмонтировать том из пода.

  - NodeUnstageVolume — очистить временные данные.

## Контейнеризация в Docker
(Для лучшего понимания, затрону немного тему контейнеризации и визуализирую ее)

Контейнеризация — это способ изолировать приложение и его зависимости в отдельную среду (контейнер).

Как устроено:
- Контейнер — это процесс, работающий на одной ОС, но с изоляцией:

  - namespace (для изоляции сетей, процессов, файловых систем)

  - cgroup (для ограничения ресурсов)

- Docker Image — это шаблон (слои файловой системы) для запуска контейнера.

- Docker Container — это запущенный экземпляр образа.

Пример процесса:
- Написания Dockerfile.

- Сборка образа командой docker build.

- Запуск контейнера командой docker run.

Контейнер запускается внутри хоста, но работает в своём namespace.

Визуализация:

<img src="https://github.com/user-attachments/assets/31560f6f-9471-453b-afc6-383af407349c" width="200" alt="Визуализация">

## Что такое под (pod)?
Pod — минимальная единица развертывания в Kubernetes.
Это группа одного или нескольких контейнеров, которые:

- разделяют одно сетевое пространство

- имеют общую файловую систему (volumes)

- управляются вместе как одна сущность.


## Что такое нода (node)?
Node — это физическая или виртуальная машина в кластере Kubernetes, где запускаются поды.

На каждой ноде работают:

- kubelet — агент, управляет подами и взаимодействует с CSI.

- kube-proxy — проксирует сетевой трафик.

- Container Runtime (обычно containerd или Docker) — запускает контейнеры.

Ноды бывают:

- Master Node — управляет кластером.

- Worker Node — запускает поды.

## Разбор файлов

#### 1. csi-driver.yaml
```yaml
apiVersion: storage.k8s.io/v1  # Версия API для CSIDriver
kind: CSIDriver                # Тип ресурса — CSI-драйвер
metadata:
  name: simple.csi.driver      # Имя драйвера (должно совпадать с `spec.csi.driver` в PV)
spec:
  attachRequired: false        # Указывает, что хранилище не требует отдельного этапа "прикрепления"
  podInfoOnMount: true         # Передавать информацию о Pod при монтировании
```
Данный манифест регистрирует CSI драйвер в kubernetes.
 

#### 2. daemonset.yaml
```yaml
apiVersion: apps/v1            # Версия API для DaemonSet
kind: DaemonSet               # Запуск пода на каждом узле кластера
metadata:
  name: simple-csi-node       # Имя DaemonSet
  namespace: kube-system      # Пространство имен для системных компонентов
spec:
  selector:
    matchLabels:
      app: simple-csi-node    # Селектор для управления подами
  template:
    metadata:
      labels:
        app: simple-csi-node  # Лейбл пода
    spec:
      containers:
        - name: simple-csi-driver
          image: f1lzz/test-simple-csi-driver:v0.1  # Образ CSI-драйвера
          imagePullPolicy: Always  # Всегда перепроверять образ в реестре и берет более свежую
          command:
            - "/app/driver"       # Команда для запуска
          volumeMounts:
            - mountPath: /csi    # Куда монтировать сокет
              name: socket-dir
      volumes:
        - name: socket-dir
          hostPath:
            path: /tmp/simple.csi.driver  # Хост-папка для сокета
            type: DirectoryOrCreate      # Создать, если не существует
```
DaemonSet гарантирует, что драйвер будет работать на каждом узле кластера.

#### 3. persistent-volume.yaml
```yaml
apiVersion: v1
kind: PersistentVolume         # Ресурс PV (физический том)
metadata:
  name: simple-pv             # Имя тома
spec:
  capacity:
    storage: 1Gi              # Размер тома
  accessModes:
    - ReadWriteOnce           # Режим доступа (только один узел)
  storageClassName: simple-storage-class  # Привязка к StorageClass
  csi:
    driver: simple.csi.driver  # Имя CSI-драйвера (должно совпадать с CSIDriver)
    volumeHandle: simple-volume-handle  # Уникальный идентификатор тома
  persistentVolumeReclaimPolicy: Retain  # Не удалять том после удаления PVC
```
PersistentVolume (PV) - это ресурс в кластере, предоставляющий физическое хранилище 

#### 4. persistent-volume-claim.yaml
```yaml
apiVersion: v1
kind: PersistentVolumeClaim    # Запрос на выделение тома (PVC)
metadata:
  name: simple-pvc            # Имя PVC
spec:
  accessModes:
    - ReadWriteOnce           # Должен совпадать с PV
  storageClassName: simple-storage-class  # Привязка к StorageClass
  resources:
    requests:
      storage: 1Gi            # Запрашиваемый размер
```
PersistentVolumeClaim (PVC) - это запрос на выделение хранилища:
  - Запрашивает 1GiB хранилища класса simple-storage-class
  - Будет привязан к PV, который соответствует этим требованиям

#### 5. storage-class.yaml
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass            # Класс хранилища
metadata:
  name: simple-storage       # Имя StorageClass
provisioner: simple.csi.driver  # Указывает, что мой драйвер будет управлять и создавать тома
volumeBindingMode: Immediate  # Привязка тома сразу после создания PVC
```
StorageClass определяет "класс" хранилища и связывает PVC c PV сразу при создании.

#### 6. test-pod.yaml
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-pod             # Тестовый под
spec:
  containers:
    - name: test-container
      image: alpine          # Образ контейнера
      command: ["sleep", "3600"]  # Простая команда для теста
      volumeMounts:
        - mountPath: /data   # Точка монтирования в контейнере
          name: simple-volume
  volumes:
    - name: simple-volume
      persistentVolumeClaim:
        claimName: simple-pvc  # Используем созданный PVC
```
Тестовый под, который использует PVC:

- Монтирует том из PVC simple-pvc в директорию /data внутри контейнера


#### 7. main.go
```go
package main

import (
	"log"
	"net"
	"os"
	"os/exec"
)

func main() {
	log.Println("Starting simple CSI driver...")

	// Путь до сокета (совпадает с volumeMount в DaemonSet)
	socketPath := "/csi/csi.sock"

	// Удаление старого сокета (если он есть)
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	// Создаем Unix-сокет для общения с kubelet
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", socketPath, err)
	}
	defer listener.Close()

	log.Printf("Listening on %s\n", socketPath)

	// Бесконечный цикл обработки запросов
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
		}
		log.Println("Got connection")

		// Эмуляция монтирования (заглушка)
		src := "/Users/f1lzz/volumes/test-volume"  # Источник (на хосте)
		target := "/Users/f1lzz/volumes/test-volume" # Цель (в контейнере)

		// Команда монтирования (nullfs — аналог bind mount(для linux))
		cmd := exec.Command("mount", "-t", "nullfs", src, target)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Mount failed: %v, output: %s", err, string(out))
		} else {
			log.Printf("Volume mounted: %s to %s", src, target)
		}

		conn.Close()  # Закрываем соединение
	}
}
```
Это очень упрощённая реализация, в реальном драйвере нужно обрабатывать все CSI RPC.

#### 8. Dockerfile
```dockerfile
FROM golang:1.24-alpine AS builder  # Сборка в Go-контейнере
WORKDIR /app
COPY go.mod .
COPY main.go .
RUN go build -o driver main.go      # Компиляция бинарника

FROM alpine                         # Минимальный образ для рантайма
WORKDIR /app
COPY --from=builder /app/driver .   # Копируем бинарник
RUN chmod +x /app/driver           # Даем права на выполнение
RUN mkdir -p /csi                  # Создаем папку для сокета
CMD ["/app/driver"]                # Запуск драйвера
```
Собирает Go-код в образ Alpine Linux:

- Этап сборки (builder) компилирует Go-приложение

- Основной этап копирует бинарник и создаёт директорию для сокета

Общая схема:

<img src="https://github.com/user-attachments/assets/fd1562e8-a00c-499c-b9b2-edcfa700595d" width="200" alt="Общая схема">



## Deploy csi-driver

#### 1. Сборка Docker-образа
```bash
docker build -t f1lzz/test-simple-csi-driver:v0.1 .
```

После сборки проверяем наш образ следующей командой
```bash
docker images
```

Ожидаемый вывод (IMAGE ID будет различаться):


| REPOSITORY    | TAG        | IMAGE ID           |  CREATED          |  SIZE  | 
| :-----------: |:----------:| :----------------: | :---------------: | :----: |
| f1lzz/test-simple-csi-driver     | v0.1         | 80e32ad3s342       | About an hour ago | 18.3MB |

#### 2. Применение манифестов (важен порядок)
```bash
kubectl apply -f csi-driver.yaml
kubectl apply -f storage-class.yaml
kubectl apply -f daemonset.yaml
kubectl apply -f persistent-volume.yaml
kubectl apply -f persistent-volume-claim.yaml
kubectl apply -f test-pod.yaml
```
#### 3. Проверка работы компонентов

Проверка DaemonSet(должен быть в статусе running):

```bash
kubectl -n kube-system get pods -l app=simple-csi-node
```

Ожидаемый вывод (NAME будет различаться):

| NAME    | READY        | STATUS           |  RESTARTS        |  AGE  | 
| :-----------: |:----------:| :----------------: | :---------------: | :----: |
| simple-csi-node-abc12     | 1/1         | Running       | 0 | 1m |


Проверка CSI-драйвера:

```bash
kubectl get csidrivers.storage.k8s.io
```

Ожидаемый вывод:

| NAME    | ATTACHREQUIRED        | PODINFOONMOUNT           |  STORAGECAPACITY       |  TOKENREQUESTS  |  REQUIRESREPUBLISH |  MODES  | AGE | 
| :-----------: |:----------:| :----------------: | :---------------: | :----: | :----: |  :----: |  :----: |
| simple.csi.driver     | false        | true       |  false | (unset) |false |Persistent |1m |


#### 4. Проверка PV и PVC
```bash
kubectl get pv
kubectl get pvc
```
PV и PVC должны быть связаны(статус Bound)

Ожидаемый вывод для pv:

| NAME    | CAPACITY        | ACCESS MODES           |  RECLAIM POLICY       |  STATUS  |  CLAIM |  STORAGECLASS  | VOLUMEATTRIBUTESCLASS | AGE | 
| :-----------: |:----------:| :----------------: | :---------------: | :----: | :----: |  :----: |  :----: |  :----: |
| simple-pv    | 1Gi       | RWO      |  Retain | Bound |default/simple-pvc | simple-storage-class |(unset) | 1m |

Ожидаемый вывод для pvc:

| NAME    | STATUS        | VOLUME        |  CAPACITY      |  ACCESS MODES |  STORAGECLASS |  VOLUMEATTRIBUTESCLASS  | AGE | 
| :-----------: |:----------:| :----------------: | :---------------: | :----: | :----: |  :----: |  :----: |
| simple-pvc    | Bound        | simple-pv      |  1Gi	| RWO  | simple-storage-class | (unset) |1m |









