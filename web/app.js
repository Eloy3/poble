const status = document.getElementById("status");
const log = document.getElementById("log");
const joinForm = document.getElementById("joinForm");
const chatForm = document.getElementById("chatForm");
const nameInput = document.getElementById("name");
const messageInput = document.getElementById("message");
const sendButton = document.getElementById("send");

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
  addLine(msg.text, msg.type === "system" ? "system" : "");
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
