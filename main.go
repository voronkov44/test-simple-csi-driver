package main

import (
	"log"
	"net"
	"os"
	"os/exec"
)

func main() {
	log.Println("Starting simple CSI driver...")

	// Путь до сокета (создается в DaemonSet)
	socketPath := "/csi/csi.sock"

	// Удаление старого сокета (если он есть)
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	// Создаем сокет
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", socketPath, err)
	}
	defer listener.Close()

	log.Printf("Listening on %s\n", socketPath)

	// Заглушка - слушаем соединения
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
		}
		log.Println("Got connection")

		// Минимальная эмуляция монтирования
		src := "/mnt/data"
		target := "/mnt/data"

		cmd := exec.Command("mount", "-t", "nullfs", src, target)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Mount failed: %v, output: %s", err, string(out))
		} else {
			log.Printf("Volume mounted: %s to %s", src, target)
		}

		conn.Close()
	}
}
