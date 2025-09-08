package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"system-monitor/handlers"
	"system-monitor/templates"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

type Server struct {
	subscriberMessageBuffer int
	subscribersMu           sync.Mutex
	subscribers             map[*Subscriber]struct{}
	app                     *fiber.App
}

type Subscriber struct {
	msgs chan []byte
	conn *websocket.Conn
}

func NewServer() *Server {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: false,
	})

	// Add logger middleware
	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))

	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	s := &Server{
		subscriberMessageBuffer: 10,
		subscribers:             make(map[*Subscriber]struct{}),
		app:                     app,
	}

	// Routes
	app.Get("/", s.indexHandler)
	app.Get("/ws", websocket.New(s.websocketHandler))

	return s
}

func (s *Server) indexHandler(c *fiber.Ctx) error {
	// Render the main page using templ
	component := templates.Index()

	// Set content type to HTML
	c.Set("Content-Type", "text/html")

	// Render the component to HTML
	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	if err != nil {
		return err
	}

	return c.SendString(buf.String())
}

func (s *Server) websocketHandler(c *websocket.Conn) {
	subscriber := &Subscriber{
		msgs: make(chan []byte, s.subscriberMessageBuffer),
		conn: c,
	}

	s.addSubscriber(subscriber)
	defer s.removeSubscriber(subscriber)

	fmt.Println("WebSocket connection established")

	// Handle incoming messages and send outgoing messages
	for {
		select {
		case msg := <-subscriber.msgs:
			err := c.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				fmt.Printf("WebSocket write error: %v\n", err)
				return
			}
		default:
			// Check if connection is still alive
			if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Printf("WebSocket ping error: %v\n", err)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (s *Server) addSubscriber(subscriber *Subscriber) {
	s.subscribersMu.Lock()
	s.subscribers[subscriber] = struct{}{}
	s.subscribersMu.Unlock()
	fmt.Printf("Added subscriber, total: %d\n", len(s.subscribers))
}

func (s *Server) removeSubscriber(subscriber *Subscriber) {
	s.subscribersMu.Lock()
	delete(s.subscribers, subscriber)
	s.subscribersMu.Unlock()
	fmt.Printf("Removed subscriber, total: %d\n", len(s.subscribers))
	close(subscriber.msgs)
}

func (s *Server) publishMsg(msg []byte) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for subscriber := range s.subscribers {
		select {
		case subscriber.msgs <- msg:
		default:
			// Channel is full, remove subscriber
			fmt.Println("Subscriber channel full, removing subscriber")
			delete(s.subscribers, subscriber)
			close(subscriber.msgs)
		}
	}
}

func (s *Server) startDataPublisher() {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Get system data
			systemInfo, err := handlers.GetSystemInfo()
			if err != nil {
				fmt.Printf("Error getting system data: %v\n", err)
				continue
			}

			// Get disk data
			diskInfo, err := handlers.GetDiskInfo()
			if err != nil {
				fmt.Printf("Error getting disk data: %v\n", err)
				continue
			}

			// Get CPU data
			cpuInfo, err := handlers.GetCPUInfo()
			if err != nil {
				fmt.Printf("Error getting CPU data: %v\n", err)
				continue
			}

			// Generate timestamp
			timeStamp := time.Now().Format("2006-01-02 15:04:05")

			// Render components to HTML
			var systemBuf, diskBuf, cpuBuf, statusBuf bytes.Buffer

			// Render system component
			systemComponent := templates.SystemData(
				systemInfo.OS,
				systemInfo.Platform,
				systemInfo.Hostname,
				systemInfo.Procs,
				systemInfo.TotalMem,
				systemInfo.FreeMem,
				systemInfo.UsedPercent,
			)
			err = systemComponent.Render(context.Background(), &systemBuf)
			if err != nil {
				fmt.Printf("Error rendering system component: %v\n", err)
				continue
			}

			// Render disk component
			diskComponent := templates.DiskData(
				diskInfo.Total,
				diskInfo.Used,
				diskInfo.Free,
				diskInfo.UsedPercent,
			)
			err = diskComponent.Render(context.Background(), &diskBuf)
			if err != nil {
				fmt.Printf("Error rendering disk component: %v\n", err)
				continue
			}

			// Render CPU component
			cpuComponent := templates.CPUData(
				cpuInfo.ModelName,
				cpuInfo.Family,
				cpuInfo.Mhz,
				cpuInfo.Percentages,
			)
			// fmt.Println("Cpu percentage: ",cpuInfo.Percentages)
			err = cpuComponent.Render(context.Background(), &cpuBuf)
			if err != nil {
				fmt.Printf("Error rendering CPU component: %v\n", err)
				continue
			}

			// Render status update component
			statusComponent := templates.StatusUpdate(timeStamp)
			err = statusComponent.Render(context.Background(), &statusBuf)
			if err != nil {
				fmt.Printf("Error rendering status component: %v\n", err)
				continue
			}

			// Create HTMX-compatible message with hx-swap-oob
			msg := []byte(fmt.Sprintf(`
				<div hx-swap-oob="innerHTML:#update-timestamp">%s</div>
				<div hx-swap-oob="innerHTML:#system-data">%s</div>
				<div hx-swap-oob="innerHTML:#cpu-data">%s</div>
				<div hx-swap-oob="innerHTML:#disk-data">%s</div>`,
					statusBuf.String(),
					systemBuf.String(),
					cpuBuf.String(),
					diskBuf.String()))

				s.publishMsg(msg)
		}
	}()
}

func main() {
	fmt.Println("🚀 Starting GOTTH System Monitor on port 6080")
	fmt.Println("📊 Stack: Go + Templ + Tailwind + HTMX")

	s := NewServer()

	// Start the data publisher goroutine
	s.startDataPublisher()

	// Start the server
	log.Fatal(s.app.Listen(":6080"))
	
}
