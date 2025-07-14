import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import './App.css';

// Define types for the grid and cell
interface CellState {
   state: boolean;
}
//
interface Grid {
  Rows: number;
  Cols: number;
  Cells: boolean[][]; // Go's `CellState` is `bool`, so use `boolean[][]` here
}
//
const API_BASE_URL = 'http://localhost:8080/api'; // Go backend API URL

function App() {
  const [grid, setGrid] = useState<Grid>({ Rows: 0, Cols: 0, Cells: [] });
  const [rows, setRows] = useState<number>(5);
  const [cols, setCols] = useState<number>(5);
  const [isGameRunning, setIsGameRunning] = useState<boolean>(false);

  const fetchGrid = useCallback(async () => {
    try {
      const response = await axios.get<Grid>(`${API_BASE_URL}/grid`);
      setGrid(response.data);
    } catch (error) {
      console.error('Error fetching grid:', error);
    }
  }, []);

  useEffect(() => {
    fetchGrid(); // Fetch initial grid on component mount
  }, [fetchGrid]);

  useEffect(() => {
    let intervalId: NodeJS.Timeout;
    if (isGameRunning) {
      intervalId = setInterval(() => {
        handleNextGeneration();
      }, 500); // Update every 500ms
    }
    return () => clearInterval(intervalId);
  }, [isGameRunning]);

  const handleSetGridSize = async () => {
    try {
      const response = await axios.post<Grid>(`${API_BASE_URL}/grid`, { rows, cols });
      setGrid(response.data);
      setIsGameRunning(false); // Stop game if grid size changes
    } catch (error) {
      console.error('Error setting grid size:', error);
      alert('Failed to set grid size. Please check input values (max 20x20).');
    }
  };

  const handleCellClick = async (rowIndex: number, colIndex: number) => {
    try {
      const newCellState = !grid.Cells[rowIndex][colIndex];
      const response = await axios.post<Grid>(`${API_BASE_URL}/cell`, {
        row: rowIndex,
        col: colIndex,
        state: newCellState,
      });
      setGrid(response.data); // Update grid with new state
    } catch (error) {
      console.error('Error toggling cell:', error);
    }
  };

  const handleResetGrid = async () => {
    try {
      const response = await axios.post<Grid>(`${API_BASE_URL}/grid/reset`);
      setGrid(response.data);
      setIsGameRunning(false); // Stop game on reset
    } catch (error) {
      console.error('Error resetting grid:', error);
    }
  };

  const handleNextGeneration = async () => {
    try {
      const response = await axios.post<Grid>(`${API_BASE_URL}/next`);
      setGrid(response.data);
    } catch (error) {
      console.error('Error getting next generation:', error);
    }
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>Conway's Game of Life</h1>
      </header>
      <div className="controls">
        <div className="grid-size-input">
          <label htmlFor="rows">Rows (Max 20):</label>
          <input
            type="number"
            id="rows"
            min="1"
            max="20"
            value={rows}
            onChange={(e) => setRows(Math.min(20, Math.max(1, parseInt(e.target.value))))}
          />
          <label htmlFor="cols">Cols (Max 20):</label>
          <input
            type="number"
            id="cols"
            min="1"
            max="20"
            value={cols}
            onChange={(e) => setCols(Math.min(20, Math.max(1, parseInt(e.target.value))))}
          />
          <button onClick={handleSetGridSize}>Set Grid Size</button>
        </div>
        <button onClick={handleResetGrid}>Reset Grid</button>
        <button onClick={() => setIsGameRunning(!isGameRunning)}>
          {isGameRunning ? 'Stop Game' : 'Start Game'}
        </button>
        <button onClick={handleNextGeneration} disabled={isGameRunning}>Next Generation</button>
      </div>
      <div className="game-grid-container">
        {grid.Cells.length > 0 ? (
          <div
            className="game-grid"
            style={{
              gridTemplateColumns: `repeat(${grid.Cols}, 20px)`,
              gridTemplateRows: `repeat(${grid.Rows}, 20px)`,
            }}
          >
            {grid.Cells.map((row, rowIndex) =>
              row.map((cellActive, colIndex) => (
                <div
                  key={`${rowIndex}-${colIndex}`}
                  className={`grid-cell ${cellActive ? 'active' : 'inactive'}`}
                  onClick={() => handleCellClick(rowIndex, colIndex)}
                ></div>
              ))
            )}
          </div>
        ) : (
          <p>Set grid size to start the game.</p>
        )}
      </div>
    </div>
  );
}

export default App;

