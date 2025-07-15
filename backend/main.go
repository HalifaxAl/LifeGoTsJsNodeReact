package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket" // Using gorilla/websocket for easy WebSocket handling
	"github.com/rs/cors"          // Using rs/cors for handling CORS
)

// CellState represents the state of a cell
type CellState bool

// Grid represents the Game of Life grid
type Grid struct {
	Rows int
	Cols int
	Cells [][]CellState
	mu sync.Mutex // Mutex to protect concurrent access to the grid
}

// NewGrid creates a new empty grid
func NewGrid(rows, cols int) *Grid {
	cells := make([][]CellState, rows)
	for i := range cells {
		cells[i] = make([]CellState, cols)
	}
	return &Grid{
		Rows:  rows,
		Cols:  cols,
		Cells: cells,
	}
}

// SetCell sets the state of a specific cell
func (g *Grid) SetCell(row, col int, state CellState) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if row >= 0 && row < g.Rows && col >= 0 && col < g.Cols {
		g.Cells[row][col] = state
	}
}

// ClearAllCells resets all cells to inactive
func (g *Grid) ClearAllCells() {
	g.mu.Lock()
	defer g.mu.Unlock()

	for r := 0; r < g.Rows; r++ {
		for c := 0; c < g.Cols; c++ {
			g.Cells[r][c] = false
		}
	}
}

// NextGeneration calculates the next generation of the grid
func (g *Grid) NextGeneration() {
	g.mu.Lock()
	defer g.mu.Unlock()

	newCells := make([][]CellState, g.Rows)
	for i := range newCells {
		newCells[i] = make([]CellState, g.Cols)
	}

	for r := 0; r < g.Rows; r++ {
		for c := 0; c < g.Cols; c++ {
			liveNeighbors := g.countLiveNeighbors(r, c)
			if g.Cells[r][c] { // Currently alive
				if liveNeighbors < 2 || liveNeighbors > 3 {
					newCells[r][c] = false // Underpopulation or Overpopulation
				} else {
					newCells[r][c] = true // Survives
				}
			} else { // Currently dead
				if liveNeighbors == 3 {
					newCells[r][c] = true // Reproduction
				} else {
					newCells[r][c] = false // Remains dead
				}
			}
		}
	}
	g.Cells = newCells
}

// countLiveNeighbors counts the live neighbors for a given cell
func (g *Grid) countLiveNeighbors(row, col int) int {
	count := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i == 0 && j == 0 {
				continue // Skip the cell itself
			}
			r, c := row+i, col+j
			if r >= 0 && r < g.Rows && c >= 0 && c < g.Cols && g.Cells[r][c] {
				count++
			}
		}
	}
	return count
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development. In production, restrict this.
		return true
	},
}

var currentGrid *Grid
var gridMux sync.Mutex // Mutex for currentGrid pointer

func main() {
	// Initialize default grid
	gridMux.Lock()
	currentGrid = NewGrid(5, 5) // Default 5x5 grid
	gridMux.Unlock()

	// Handle CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"}, // Adjust this to your frontend's URL
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		AllowCredentials: true,
	})

	http.Handle("/ws", c.Handler(http.HandlerFunc(handleWebSocket)))
	http.Handle("/api/grid", c.Handler(http.HandlerFunc(handleGrid)))
	http.Handle("/api/grid/reset", c.Handler(http.HandlerFunc(handleResetGrid)))
	http.Handle("/api/cell", c.Handler(http.HandlerFunc(handleCellToggle)))
	http.Handle("/api/next", c.Handler(http.HandlerFunc(handleNextGeneration)))

	port := ":8080"
	fmt.Printf("Go server listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	// Send initial grid state
	gridMux.Lock()
	initialGridBytes, _ := json.Marshal(currentGrid)
	gridMux.Unlock()
	conn.WriteMessage(websocket.TextMessage, initialGridBytes)

	// Keep connection open to send updates
	for {
		// This loop can be used to receive messages from the client if needed,
		// but for this Game of Life, updates are primarily server-to-client.
		// For now, just keep the connection alive.
		time.Sleep(5 * time.Second) // Keep alive, adjust as needed
	}
}

func handleGrid(w http.ResponseWriter, r *http.Request) {
	gridMux.Lock()
	defer gridMux.Unlock()

	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentGrid)
	case "POST":
		var requestBody struct {
			Rows int `json:"rows"`
			Cols int `json:"cols"`
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if requestBody.Rows <= 0 || requestBody.Cols <= 0 || requestBody.Rows > 20 || requestBody.Cols > 20 {
			http.Error(w, "Invalid grid dimensions. Max 20x20.", http.StatusBadRequest)
			return
		}

		currentGrid = NewGrid(requestBody.Rows, requestBody.Cols)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(currentGrid)
	default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleResetGrid(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	gridMux.Lock()
	defer gridMux.Unlock()
	currentGrid.ClearAllCells()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(currentGrid) // Send updated grid
}

func handleCellToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		Row   int  `json:"row"`
		Col   int  `json:"col"`
		State bool `json:"state"` // true for active, false for inactive
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gridMux.Lock()
	defer gridMux.Unlock()

	currentGrid.SetCell(requestBody.Row, requestBody.Col, CellState(requestBody.State))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(currentGrid) // Send updated grid
}

func handleNextGeneration(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	gridMux.Lock()
	defer gridMux.Unlock()
	currentGrid.NextGeneration()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(currentGrid) // Send updated grid
}


