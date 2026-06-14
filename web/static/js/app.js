// app.js - Core utilities

function escapeHTML(str) {
  if (typeof str !== 'string') return '';
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

// Simple Markdown renderer (safe, no raw HTML)
function renderMarkdown(text) {
  if (!text) return '';
  let escaped = escapeHTML(text);

  // Bold: **text** or __text__
  escaped = escaped.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  escaped = escaped.replace(/__(.+?)__/g, '<strong>$1</strong>');

  // Italic: *text* or _text_
  escaped = escaped.replace(/\*([^*]+?)\*/g, '<em>$1</em>');

  // Numbered list items: lines starting with "1. " etc
  escaped = escaped.replace(/^(\d+)\.\s+(.+)$/gm, '<div class="flex gap-2 mb-1"><span class="text-[#2E7D32] font-semibold min-w-[1.2em]">$1.</span><span>$2</span></div>');

  // Bullet points: lines starting with "- " or "• "
  escaped = escaped.replace(/^[-•]\s+(.+)$/gm, '<div class="flex gap-2 mb-1"><span class="text-[#4CAF50] mt-1">•</span><span>$1</span></div>');

  // Line breaks
  escaped = escaped.replace(/\n\n/g, '</p><p class="mb-2">');
  escaped = escaped.replace(/\n/g, '<br>');

  return '<p class="mb-2">' + escaped + '</p>';
}

function openSidebar() {
  document.getElementById('sidebar').classList.remove('-translate-x-full');
  document.getElementById('sidebar-overlay').classList.remove('hidden');
}

function closeSidebar() {
  document.getElementById('sidebar').classList.add('-translate-x-full');
  document.getElementById('sidebar-overlay').classList.add('hidden');
}

function autoResizeTextarea(el) {
  el.style.height = 'auto';
  el.style.height = Math.min(el.scrollHeight, 128) + 'px';

  const len = el.value.length;
  const counter = document.getElementById('char-count');
  if (len > 3000) {
    counter.classList.remove('hidden');
    counter.textContent = `${len}/4000`;
  } else {
    counter.classList.add('hidden');
  }
}

function handleInputKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    sendMessage();
  }
}

function scrollToBottom() {
  const container = document.getElementById('chat-container');
  container.scrollTop = container.scrollHeight;
}
