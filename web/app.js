const status = document.getElementById("status");
const log = document.getElementById("log");
const joinForm = document.getElementById("joinForm");
const chatForm = document.getElementById("chatForm");
const nameInput = document.getElementById("name");
const messageInput = document.getElementById("message");
const sendButton = document.getElementById("send");
const readyButton = document.getElementById("ready");
const startGameButton = document.getElementById("startGame");

const protocol = location.protocol === "https:" ? "wss" : "ws";
const socket = new WebSocket(`${protocol}://${location.host}/ws`);

function addLine(text, className = "") {
  const div = document.createElement("div");
  div.className = `line ${className}`;
  div.textContent = text;
  log.appendChild(div);
  log.scrollTop = log.scrollHeight;
}

socket.addEventListener("open", () => {
  status.textContent = "Connected";
  addLine("Connected to the game server.", "system");
});

socket.addEventListener("close", () => {
  status.textContent = "Disconnected";
  messageInput.disabled = true;
  sendButton.disabled = true;
  addLine("Connection closed.", "system");
});

socket.addEventListener("message", (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === "game_started") {
    readyButton.disabled = true;
    startGameButton.disabled = true;
  }

  if (msg.type === "private_role") {
    const role = msg.state;
    addLine(`${msg.text} Team: ${role.team}. Ability: ${role.ability}.`, "role");
    return;
  }

  if (msg.text) {
    addLine(msg.text, msg.type === "system" ? "system" : "");
  }
});

joinForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const name = nameInput.value.trim();
  if (!name || socket.readyState !== WebSocket.OPEN) return;

  socket.send(JSON.stringify({
    type: "join",
    text: name
  }));

  nameInput.disabled = true;
  joinForm.querySelector("button").disabled = true;
  messageInput.disabled = false;
  sendButton.disabled = false;
  readyButton.disabled = false;
  startGameButton.disabled = false;
  messageInput.focus();
});

chatForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const text = messageInput.value.trim();
  if (!text || socket.readyState !== WebSocket.OPEN) return;

  socket.send(JSON.stringify({
    type: "chat",
    text
  }));

  messageInput.value = "";
  messageInput.focus();
});

readyButton.addEventListener("click", () => {
  if (socket.readyState !== WebSocket.OPEN) return;
  socket.send(JSON.stringify({ type: "ready" }));
  readyButton.disabled = true;
});

startGameButton.addEventListener("click", () => {
  if (socket.readyState !== WebSocket.OPEN) return;
  socket.send(JSON.stringify({ type: "start_game" }));
});
