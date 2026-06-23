// assistant.js - AgriConnect AI chat interface

const State = {
  conversationId: null,
  language: 'english',
  district: '',
  crop: '',
  sending: false,
  _weatherFetched: false,
};

// ─── Initialization ────────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', () => {
  const assistantRoot = document.querySelector("[data-assistant-root]");
  if (!assistantRoot) return;

  loadConversations();

  // Restore last state from sessionStorage
  const saved = sessionStorage.getItem('agri_state');
  if (saved) {
    try {
      const s = JSON.parse(saved);
      if (s.language) setLanguage(s.language, false);
      if (s.district) {
        State.district = s.district;
        document.getElementById('district-select').value = s.district;
      }
      if (s.crop) {
        State.crop = s.crop;
        document.getElementById('crop-select').value = s.crop;
      }
    } catch {}
  }

  // If user has a saved district from registration, auto-fetch weather once
  const userDistrict = window.AGRI_CONFIG?.userDistrict;
  if (userDistrict && !State.district) {
    State.district = userDistrict;
    const sel = document.getElementById('district-select');
    if (sel) sel.value = userDistrict;
    saveState();
  }

  // Preserve user district even if not saved to session
  if (userDistrict && !saved) {
    setDistrict(userDistrict);
  }

  // Reload last conversation if present
  const lastConv = sessionStorage.getItem('agri_conv');
  if (lastConv) {
    loadConversation(lastConv).catch(function() {
      sessionStorage.removeItem('agri_conv');
    });
  }

  // Auto-weather on page load (once)
  if (State.district && !State._weatherFetched) {
    State._weatherFetched = true;
    fetchWeather(true);
  }
});

function saveState() {
  sessionStorage.setItem('agri_state', JSON.stringify({
    language: State.language,
    district: State.district,
    crop: State.crop,
  }));
}

// ─── Settings ──────────────────────────────────────────────────────────────

function setLanguage(lang, persist = true) {
  State.language = lang;

  document.querySelectorAll('.lang-btn').forEach(b => b.classList.remove('active-lang'));
  document.getElementById('lang-' + lang)?.classList.add('active-lang');

  // Show relevant suggestions
  const enDiv = document.getElementById('en-suggestions');
  const krDiv = document.getElementById('krio-suggestions');
  if (enDiv && krDiv) {
    if (lang === 'krio') {
      enDiv.classList.add('hidden');
      krDiv.classList.remove('hidden');
    } else {
      enDiv.classList.remove('hidden');
      krDiv.classList.add('hidden');
    }
  }

  // Update input placeholder
  const input = document.getElementById('message-input');
  if (input) {
    input.placeholder = lang === 'krio'
      ? 'Aks agri question ya...'
      : 'Ask an agricultural question...';
  }

  if (persist) saveState();
}

function setDistrict(value) {
  if (value === State.district) return;
  State.district = value;
  State._weatherFetched = false;
  saveState();
  document.getElementById('weather-panel')?.classList.add('hidden');
  fetchWeather(true);
}

function setCrop(value) {
  State.crop = value;
  saveState();
}

// ─── Conversations ─────────────────────────────────────────────────────────

async function loadConversations() {
  try {
    const res = await fetch('/api/v1/conversations');
    if (!res.ok) return;
    const convs = await res.json();
    renderConversationList(convs);
  } catch (e) {
    console.error('Failed to load conversations:', e);
  }
}

function renderConversationList(convs) {
  const container = document.getElementById('conversations-container');
  if (!convs || convs.length === 0) {
    container.innerHTML = '<p class="text-xs text-[#6B7280] px-2 py-4 text-center">No conversations yet</p>';
    return;
  }

  container.innerHTML = convs.map(c => `
    <div class="conversation-item ${c.id === State.conversationId ? 'active' : ''}"
         onclick="loadConversation('${escapeHTML(c.id)}')"
         data-id="${escapeHTML(c.id)}">
      <i data-lucide="message-circle" class="w-3.5 h-3.5 flex-shrink-0"></i>
      <span class="truncate flex-1">${escapeHTML(c.title)}</span>
      <button onclick="deleteConversation(event, '${escapeHTML(c.id)}')" class="p-0.5 hover:text-red-500 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
        <i data-lucide="trash-2" class="w-3 h-3"></i>
      </button>
    </div>
  `).join('');

  if (typeof lucide !== 'undefined') lucide.createIcons();
}

async function startNewConversation() {
  if (!State.district) {
    showToast('Please select a district before starting a conversation.', 'warning');
    return;
  }

  try {
    const res = await fetch('/api/v1/conversations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        preferred_language: State.language,
        district: State.district,
        crop: State.crop,
      }),
    });

    if (!res.ok) throw new Error('Failed to create conversation');
    const conv = await res.json();

    State.conversationId = conv.id;
    sessionStorage.setItem('agri_conv', conv.id);

    // Clear messages
    document.getElementById('messages-area').innerHTML = '';
    document.getElementById('messages-area').classList.remove('hidden');
    document.getElementById('welcome-screen').classList.add('hidden');

    await loadConversations();
    closeSidebar();
    document.getElementById('message-input').focus();
  } catch (e) {
    showToast('Failed to start conversation. Please try again.', 'error');
  }
}

async function loadConversation(id) {
  try {
    const res = await fetch(`/api/v1/conversations/${id}`);
    if (!res.ok) throw new Error('Not found');
    const conv = await res.json();

    State.conversationId = conv.id;
    sessionStorage.setItem('agri_conv', conv.id);

    // Restore conversation settings
    if (conv.preferred_language) setLanguage(conv.preferred_language, false);
    if (conv.district) {
      State.district = conv.district;
      document.getElementById('district-select').value = conv.district;
    }
    if (conv.crop) {
      State.crop = conv.crop;
      document.getElementById('crop-select').value = conv.crop;
    }

    // Show messages
    const messagesArea = document.getElementById('messages-area');
    messagesArea.innerHTML = '';
    document.getElementById('welcome-screen').classList.add('hidden');
    messagesArea.classList.remove('hidden');

    if (conv.messages) {
      conv.messages.forEach(m => {
        if (m.role === 'user' || m.role === 'assistant') {
          appendMessage(m.role, m.content, m.id);
        }
      });
    }

    // Update sidebar active state
    document.querySelectorAll('.conversation-item').forEach(el => {
      el.classList.toggle('active', el.dataset.id === id);
    });

    scrollToBottom();
    closeSidebar();
    document.getElementById('message-input').focus();
  } catch (e) {
    showToast('Failed to load conversation.', 'error');
  }
}

async function deleteConversation(event, id) {
  event.stopPropagation();
  if (!confirm('Delete this conversation?')) return;

  try {
    const res = await fetch(`/api/v1/conversations/${id}`, { method: 'DELETE' });
    if (!res.ok) throw new Error('Delete failed');

    if (State.conversationId === id) {
      State.conversationId = null;
      sessionStorage.removeItem('agri_conv');
      document.getElementById('messages-area').classList.add('hidden');
      document.getElementById('welcome-screen').classList.remove('hidden');
    }
    await loadConversations();
  } catch (e) {
    showToast('Failed to delete conversation.', 'error');
  }
}

// ─── Messaging ─────────────────────────────────────────────────────────────

async function sendMessage() {
  if (State.sending) return;

  const input = document.getElementById('message-input');
  const text = input.value.trim();
  if (!text) return;

  if (text.length < 2) {
    showToast('Message is too short.', 'warning');
    return;
  }

  // Auto-create conversation if none exists
  if (!State.conversationId) {
    if (!State.district) {
      showToast('Please select a district first.', 'warning');
      return;
    }
    await startNewConversation();
    if (!State.conversationId) return;
  }

  input.value = '';
  autoResizeTextarea(input);

  State.sending = true;
  setSendingState(true);

  appendMessage('user', text);

  const typingId = appendTypingIndicator();

  try {
    await streamMessage(State.conversationId, text, typingId);
  } catch (e) {
    removeTypingIndicator(typingId);
    appendErrorMessage('Failed to get a response. Please try again.');
  } finally {
    State.sending = false;
    setSendingState(false);
    hideStatus();
    await loadConversations();
  }
}

async function streamMessage(convId, text, typingId) {
  const res = await fetch(`/api/v1/conversations/${convId}/messages/stream`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message: text }),
  });

  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.error || `HTTP ${res.status}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let assistantContent = '';
  let msgId = null;

  // Replace typing indicator with empty bubble
  removeTypingIndicator(typingId);
  const bubbleId = 'bubble-' + Date.now();
  appendMessage('assistant', '', null, bubbleId);
  const bubble = document.getElementById(bubbleId);

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop();

    for (const line of lines) {
      if (!line.trim()) continue;

      if (line.startsWith('event: ')) {
        // Event type handled with next data line
        continue;
      }

      if (line.startsWith('data: ')) {
        const prevLine = lines[lines.indexOf(line) - 1] || '';
        const eventType = prevLine.startsWith('event: ') ? prevLine.slice(7) : 'token';

        try {
          const payload = JSON.parse(line.slice(6));

          if (eventType === 'token' || payload.text !== undefined) {
            assistantContent += payload.text || '';
            if (bubble) {
              bubble.innerHTML = renderMarkdown(assistantContent);
              scrollToBottom();
            }
          } else if (eventType === 'status' || payload.message) {
            if (!payload.text && !payload.message_id) {
              showStatus(payload.message || '');
            }
          } else if (payload.message_id) {
            msgId = payload.message_id;
            hideStatus();
          } else if (payload.message && payload.message.includes('error')) {
            appendErrorMessage(payload.message);
          }
        } catch {}
      }
    }
  }
}

// Parse SSE properly
async function streamMessageSSE(convId, text, typingId) {
  const res = await fetch(`/api/v1/conversations/${convId}/messages/stream`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ message: text }),
  });

  if (!res.ok) throw new Error(`HTTP ${res.status}`);

  removeTypingIndicator(typingId);
  const bubbleId = 'bubble-' + Date.now();
  appendMessage('assistant', '', null, bubbleId);
  const bubble = document.getElementById(bubbleId);

  let assistantContent = '';
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let currentEvent = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() ?? '';

    for (const line of lines) {
      if (line.startsWith('event: ')) {
        currentEvent = line.slice(7).trim();
      } else if (line.startsWith('data: ')) {
        try {
          const payload = JSON.parse(line.slice(6));
          if (currentEvent === 'token') {
            assistantContent += payload.text || '';
            if (bubble) {
              bubble.innerHTML = renderMarkdown(assistantContent);
              scrollToBottom();
            }
          } else if (currentEvent === 'status') {
            showStatus(payload.message || '');
          } else if (currentEvent === 'complete') {
            hideStatus();
          } else if (currentEvent === 'error') {
            if (bubble) bubble.innerHTML = `<span class="text-red-500">${escapeHTML(payload.message)}</span>`;
          }
        } catch {}
      }
    }
  }
}

function sendSuggestion(btn) {
  const input = document.getElementById('message-input');
  const text = btn.textContent.trim();
  // Remove icon text if any
  input.value = text.replace(/^[^\w]+/, '');
  sendMessage();
}

// ─── UI Helpers ────────────────────────────────────────────────────────────

function appendMessage(role, content, id, bubbleId) {
  const area = document.getElementById('messages-area');
  area.classList.remove('hidden');
  document.getElementById('welcome-screen').classList.add('hidden');

  const isUser = role === 'user';
  const wrapper = document.createElement('div');
  wrapper.className = isUser ? 'chat-message-user' : 'chat-message-assistant';
  if (id) wrapper.dataset.messageId = id;

  const bubble = document.createElement('div');
  bubble.className = isUser ? 'message-bubble-user' : 'message-bubble-assistant';
  if (bubbleId) bubble.id = bubbleId;

  if (isUser) {
    bubble.textContent = content;
  } else {
    bubble.innerHTML = content ? renderMarkdown(content) : '';
  }

  if (!isUser) {
    const avatar = document.createElement('div');
    avatar.className = 'w-7 h-7 rounded-full bg-[#2E7D32] flex items-center justify-center flex-shrink-0 mt-1';
    avatar.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z"/></svg>';
    wrapper.appendChild(avatar);
  }

  wrapper.appendChild(bubble);
  area.appendChild(wrapper);
  scrollToBottom();

  return bubble;
}

function appendTypingIndicator() {
  const area = document.getElementById('messages-area');
  const id = 'typing-' + Date.now();
  const wrapper = document.createElement('div');
  wrapper.className = 'chat-message-assistant';
  wrapper.id = id;
  wrapper.innerHTML = `
    <div class="w-7 h-7 rounded-full bg-[#2E7D32] flex items-center justify-center flex-shrink-0 mt-1">
      <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z"/></svg>
    </div>
    <div class="message-bubble-assistant">
      <div class="flex gap-1 items-center py-1">
        <div class="w-2 h-2 bg-[#4CAF50] rounded-full animate-bounce" style="animation-delay:0ms"></div>
        <div class="w-2 h-2 bg-[#4CAF50] rounded-full animate-bounce" style="animation-delay:150ms"></div>
        <div class="w-2 h-2 bg-[#4CAF50] rounded-full animate-bounce" style="animation-delay:300ms"></div>
      </div>
    </div>
  `;
  area.appendChild(wrapper);
  scrollToBottom();
  return id;
}

function removeTypingIndicator(id) {
  document.getElementById(id)?.remove();
}

function appendErrorMessage(text) {
  const area = document.getElementById('messages-area');
  const wrapper = document.createElement('div');
  wrapper.className = 'flex justify-center';
  wrapper.innerHTML = `
    <div class="bg-red-50 border border-red-200 rounded-xl px-4 py-3 text-sm text-red-700 flex items-center gap-2 max-w-sm">
      <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
      <span>${escapeHTML(text)}</span>
    </div>
  `;
  area.appendChild(wrapper);
  scrollToBottom();
}

function setSendingState(sending) {
  const btn = document.getElementById('send-btn');
  const input = document.getElementById('message-input');
  btn.disabled = sending;
  input.disabled = sending;
}

function showStatus(message) {
  const bar = document.getElementById('status-bar');
  const text = document.getElementById('status-text');
  bar.classList.remove('hidden');
  text.textContent = message;
}

function hideStatus() {
  document.getElementById('status-bar').classList.add('hidden');
}

function showToast(message, type = 'info') {
  const colors = {
    info: 'bg-blue-50 border-blue-200 text-blue-700',
    warning: 'bg-amber-50 border-amber-200 text-amber-700',
    error: 'bg-red-50 border-red-200 text-red-700',
    success: 'bg-green-50 border-green-200 text-green-700',
  };

  const toast = document.createElement('div');
  toast.className = `fixed top-4 right-4 z-50 px-4 py-3 rounded-xl border text-sm shadow-lg max-w-sm ${colors[type] || colors.info}`;
  toast.textContent = message;
  document.body.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transition = 'opacity 0.3s';
    setTimeout(() => toast.remove(), 300);
  }, 3500);
}

// ─── Weather ───────────────────────────────────────────────────────────────

async function fetchWeather(silent) {
  if (!State.district) {
    showToast('Please select a district first.', 'warning');
    return;
  }

  const btn = document.getElementById('weather-btn');
  const panel = document.getElementById('weather-panel');
  const content = document.getElementById('weather-content');

  if (!silent) {
    btn.innerHTML = '<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg><span>Loading...</span>';
  }

  if (!silent) {
    content.innerHTML = `
      <div class="flex items-center gap-3 py-2">
        <div class="w-4 h-4 border-2 border-[#2E7D32] border-t-transparent rounded-full animate-spin"></div>
        <span class="text-sm text-[#6B7280]">Loading weather...</span>
      </div>`;
    panel.classList.remove('hidden');
  }

  try {
    const res = await fetch(`/api/v1/weather?district=${encodeURIComponent(State.district)}`);
    if (!res.ok) {
      if (res.status === 429) {
        throw { message: 'Too many weather requests. Please wait briefly and retry.', status: 429 };
      } else if (res.status === 502 || res.status === 503) {
        throw { message: 'The weather provider is temporarily unavailable.', status: res.status };
      }
      throw { message: 'Unable to load weather data.', status: res.status };
    }
    const data = await res.json();
    renderWeatherPanel(data);
  } catch (e) {
    const msg = e.message || e.status || 'Check your internet connection and retry.';
    if (silent) {
      content.innerHTML = `
        <div class="flex items-center justify-between">
          <div class="text-sm text-[#6B7280]">
            <span class="font-semibold">${escapeHTML(State.district)}</span>
          </div>
          <button onclick="fetchWeather()" class="text-xs text-[#2E7D32] hover:underline font-medium">Retry</button>
        </div>
        <p class="text-xs text-red-600 mt-1">${escapeHTML(msg)}</p>`;
    } else {
      showToast(msg, 'error');
    }
  } finally {
    btn.innerHTML = '<i data-lucide="cloud-sun" class="w-4 h-4 text-[#FFC107]"></i><span>Check District Weather</span>';
    if (typeof lucide !== 'undefined') lucide.createIcons();
  }
}

function renderWeatherPanel(data) {
  const panel = document.getElementById('weather-panel');
  const content = document.getElementById('weather-content');

  const today = data.daily && data.daily[0];
  const cached = data.cached ? ' (cached)' : '';

  content.innerHTML = `
    <div class="flex items-center justify-between mb-2">
      <div class="flex items-center gap-2">
        <svg class="w-4 h-4 text-[#FFC107]" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"/></svg>
        <span class="font-semibold text-sm text-[#1F2937]">${escapeHTML(data.district)}, Sierra Leone${cached}</span>
      </div>
      <button onclick="document.getElementById('weather-panel').classList.add('hidden')" class="text-[#6B7280] hover:text-[#1F2937] text-lg leading-none">&times;</button>
    </div>
    <div class="flex gap-4 flex-wrap">
      <div class="weather-badge"><svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"/></svg>${data.current.temperature_c}°C</div>
      <div class="weather-badge">💧 ${data.current.humidity_percent}% humidity</div>
      <div class="weather-badge">🌧 ${data.current.precipitation_mm}mm now</div>
      <div class="weather-badge">💨 ${data.current.wind_speed_kmh} km/h</div>
      ${today ? `<div class="weather-badge">☁️ ${today.rain_probability_percent}% rain today</div>` : ''}
    </div>
  `;

  panel.classList.remove('hidden');
}
