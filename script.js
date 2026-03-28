const boardElement = document.getElementById('board');
const messageElement = document.getElementById('message');
const errorElement = document.getElementById('error');
const resetButton = document.getElementById('reset-button');

const BOARD_SIZE = 8;
const EMPTY = 0;
const BLACK = 1;
const WHITE = 2;

const CPU_PLAYER = WHITE;
const HUMAN_PLAYER = BLACK;

let board = [];
let currentPlayer;

// ---- Game initialization ---------------------------------------------------

function initializeGame() {
    board = Array(BOARD_SIZE).fill(0).map(() => Array(BOARD_SIZE).fill(EMPTY));
    board[3][3] = WHITE;
    board[3][4] = BLACK;
    board[4][3] = BLACK;
    board[4][4] = WHITE;
    currentPlayer = HUMAN_PLAYER;
    clearError();
    drawBoard();
    updateMessage();
}

// ---- Board rendering -------------------------------------------------------

function drawBoard() {
    boardElement.innerHTML = '';
    const hints = new Set(
        currentPlayer === HUMAN_PLAYER
            ? getValidMoves(HUMAN_PLAYER).map(m => `${m.r},${m.c}`)
            : []
    );

    for (let r = 0; r < BOARD_SIZE; r++) {
        for (let c = 0; c < BOARD_SIZE; c++) {
            const cell = document.createElement('div');
            cell.classList.add('cell');
            cell.dataset.row = r;
            cell.dataset.col = c;
            cell.addEventListener('click', handleCellClick);

            if (board[r][c] === EMPTY && hints.has(`${r},${c}`)) {
                cell.classList.add('hint');
            }

            if (board[r][c] !== EMPTY) {
                const piece = document.createElement('div');
                piece.classList.add('piece', board[r][c] === BLACK ? 'black' : 'white');
                cell.appendChild(piece);
            }

            boardElement.appendChild(cell);
        }
    }
}

function updateMessage(extra = '') {
    const b = countPieces(BLACK);
    const w = countPieces(WHITE);
    const turn = currentPlayer === BLACK ? '黒 (あなた)' : '白 (LLM)';
    messageElement.textContent = `黒: ${b}  白: ${w}　|　${turn}の番${extra}`;
}

function showError(msg) {
    errorElement.textContent = msg;
}

function clearError() {
    errorElement.textContent = '';
}

// ---- Game logic ------------------------------------------------------------

function countPieces(player) {
    let count = 0;
    for (let r = 0; r < BOARD_SIZE; r++)
        for (let c = 0; c < BOARD_SIZE; c++)
            if (board[r][c] === player) count++;
    return count;
}

function getValidMoves(player) {
    const moves = [];
    for (let r = 0; r < BOARD_SIZE; r++)
        for (let c = 0; c < BOARD_SIZE; c++)
            if (isValidMove(r, c, player)) moves.push({ r, c });
    return moves;
}

function isValidMove(row, col, player) {
    if (board[row][col] !== EMPTY) return false;
    const opponent = player === BLACK ? WHITE : BLACK;

    for (let dr = -1; dr <= 1; dr++) {
        for (let dc = -1; dc <= 1; dc++) {
            if (dr === 0 && dc === 0) continue;
            let r = row + dr, c = col + dc;
            let foundOpponent = false;
            while (r >= 0 && r < BOARD_SIZE && c >= 0 && c < BOARD_SIZE && board[r][c] === opponent) {
                r += dr; c += dc; foundOpponent = true;
            }
            if (foundOpponent && r >= 0 && r < BOARD_SIZE && c >= 0 && c < BOARD_SIZE && board[r][c] === player)
                return true;
        }
    }
    return false;
}

function placePieceAndFlip(row, col, player) {
    board[row][col] = player;
    const opponent = player === BLACK ? WHITE : BLACK;

    for (let dr = -1; dr <= 1; dr++) {
        for (let dc = -1; dc <= 1; dc++) {
            if (dr === 0 && dc === 0) continue;
            let r = row + dr, c = col + dc;
            const toFlip = [];
            while (r >= 0 && r < BOARD_SIZE && c >= 0 && c < BOARD_SIZE && board[r][c] === opponent) {
                toFlip.push({ r, c }); r += dr; c += dc;
            }
            if (r >= 0 && r < BOARD_SIZE && c >= 0 && c < BOARD_SIZE && board[r][c] === player)
                toFlip.forEach(p => { board[p.r][p.c] = player; });
        }
    }
}

// ---- Turn management -------------------------------------------------------

function handleCellClick(event) {
    if (currentPlayer === CPU_PLAYER) return;
    const row = parseInt(event.currentTarget.dataset.row);
    const col = parseInt(event.currentTarget.dataset.col);

    if (!isValidMove(row, col, HUMAN_PLAYER)) {
        showError('そこには置けません');
        return;
    }
    clearError();
    placePieceAndFlip(row, col, HUMAN_PLAYER);
    advanceTurn();
}

function advanceTurn() {
    currentPlayer = currentPlayer === BLACK ? WHITE : BLACK;
    drawBoard();

    // Pass check
    if (getValidMoves(currentPlayer).length === 0) {
        const other = currentPlayer === BLACK ? WHITE : BLACK;
        if (getValidMoves(other).length === 0) {
            endGame();
            return;
        }
        updateMessage(' — パス！');
        currentPlayer = other;
        drawBoard();
        updateMessage();
    }

    updateMessage();

    if (currentPlayer === CPU_PLAYER) {
        updateMessage(' — 考え中…');
        setTimeout(makeCpuMove, 500);
    }
}

// ---- LLM move --------------------------------------------------------------

async function makeCpuMove() {
    const validMoves = getValidMoves(CPU_PLAYER);
    if (validMoves.length === 0) {
        advanceTurn();
        return;
    }

    let chosenMove = null;
    try {
        const response = await fetch('/api/move', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ board, validMoves }),
        });
        if (!response.ok) throw new Error(`server error ${response.status}`);
        chosenMove = await response.json();
    } catch (err) {
        console.error('move API error:', err);
        showError('LLMとの通信でエラーが発生しました (ランダムな手を選択)');
    }

    // Validate server response
    const valid = chosenMove && validMoves.find(m => m.r === chosenMove.row && m.c === chosenMove.col);
    if (valid) {
        placePieceAndFlip(valid.r, valid.c, CPU_PLAYER);
    } else {
        const random = validMoves[Math.floor(Math.random() * validMoves.length)];
        placePieceAndFlip(random.r, random.c, CPU_PLAYER);
    }

    advanceTurn();
}

// ---- Game end --------------------------------------------------------------

function endGame() {
    const b = countPieces(BLACK);
    const w = countPieces(WHITE);
    const result = b > w ? '黒 (あなた) の勝ち！' : w > b ? '白 (LLM) の勝ち！' : '引き分け！';
    messageElement.textContent = `ゲーム終了  黒: ${b}  白: ${w}  —  ${result}`;
    boardElement.querySelectorAll('.cell').forEach(cell => {
        cell.replaceWith(cell.cloneNode(true)); // remove all click listeners
    });
}

// ---- Bootstrap -------------------------------------------------------------

resetButton.addEventListener('click', initializeGame);
initializeGame();
