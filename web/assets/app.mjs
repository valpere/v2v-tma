/**
 * v2v-tma client: Telegram WebApp SDK, text/voice, server-owned cart, Stars checkout.
 * Pure helpers live in core.mjs (unit-tested), microphone capture in recorder.mjs.
 */
import { esc, qtyOf as qtyIn, plural, langOf, makeApi, b64ToBytes } from "./core.mjs";
import * as recorder from "./recorder.mjs";

  // 1. Ініціалізація Telegram WebApp
  const tg = window.Telegram?.WebApp;
  if (tg) {
    tg.ready();
    tg.expand();
    // Налаштування кольору заголовка під тему
    if (tg.setHeaderColor) tg.setHeaderColor('#0f172a');
    if (tg.setBackgroundColor) tg.setBackgroundColor('#020617');
  }

  // 2. Стан додатку
  const state = {
    domainId: null, // nothing is chosen until the user picks a topic
    domains: {},
    sessionId: null,
    cart: { items: [], count: 0, total_uah: 0, total_stars: 0 }, // authoritative copy from the server
    voiceMode: false, // spoken replies: false = text only (free), true = the engine's voice; the mic needs no toggle
    isRecording: false,
    mediaRecorder: null,
    audioChunks: []
  };

  // 3. Елементи DOM
  const dom = {
    topicBtn: document.getElementById('topic-btn'),
    domainName: document.getElementById('domain-name'),
    picker: document.getElementById('topic-picker'),
    pickerGrid: document.getElementById('topic-grid'),
    pickerStatus: document.getElementById('topic-picker-status'),
    view: document.getElementById('concierge-view'),
    bottomBar: document.getElementById('bottom-bar'),
    topBar: document.getElementById('top-bar'),
    domainIcon: document.getElementById('domain-icon'),
    domainTagline: document.getElementById('domain-tagline'),
    conciergeMessage: document.getElementById('concierge-message'),
    quickPrompts: document.getElementById('quick-prompts-container'),
    catalogItems: document.getElementById('catalog-items-container'),
    itemsCountBadge: document.getElementById('items-count-badge'),
    textInput: document.getElementById('text-input'),
    sendBtn: document.getElementById('send-btn'),
    micBtn: document.getElementById('mic-btn'),
    micIcon: document.getElementById('mic-icon'),
    voiceToggle: document.getElementById('voice-mode-toggle'),
    voiceLabel: document.getElementById('voice-mode-label'),
    cartBar: document.getElementById('cart-bar'),
    cartSummaryText: document.getElementById('cart-summary-text'),
    cartSubtotalText: document.getElementById('cart-subtotal-text'),
    checkoutBtn: document.getElementById('checkout-btn'),
    checkoutStarsBadge: document.getElementById('checkout-stars-badge'),
    checkoutModal: document.getElementById('checkout-modal'),
    closeModalBtn: document.getElementById('close-modal-btn'),
    modalCartItems: document.getElementById('modal-cart-items'),
    modalTotalUAH: document.getElementById('modal-total-uah'),
    modalTotalStars: document.getElementById('modal-total-stars'),
    payStarsBtn: document.getElementById('pay-stars-btn'),
    waveBars: document.querySelectorAll('.wave-bar')
  };

  // 4. Допоміжний Haptic Feedback
  function triggerHaptic(type = 'light') {
    if (tg?.HapticFeedback) {
      if (type === 'success' || type === 'error' || type === 'warning') {
        tg.HapticFeedback.notificationOccurred(type);
      } else {
        tg.HapticFeedback.impactOccurred(type);
      }
    }
  }

  // every call carries the Telegram initData; the server enforces it
  const api = makeApi(window.fetch.bind(window), tg?.initData || "");
  const LANG = langOf(tg?.initDataUnsafe?.user?.language_code);

  function notify(msg) {
    if (tg?.showAlert) tg.showAlert(msg); else alert(msg);
  }

  // 5. Завантаження доменів та каталогу
  async function loadDomains() {
    try {
      const data = await api.json('GET', '/v1/topics');
      data.forEach(d => state.domains[d.id] = d);
      renderPicker(data);
    } catch (err) {
      dom.pickerStatus.textContent = 'Сервер недоступний. Спробуйте пізніше.';
      console.warn('API error:', err);
    }
  }

  // The first screen: one card per topic, so nothing is hidden behind a dropdown.
  function renderPicker(topics) {
    dom.pickerGrid.innerHTML = '';
    topics.forEach(d => {
      const card = document.createElement('button');
      card.type = 'button';
      card.className = 'text-left bg-slate-900/90 border border-slate-800 hover:border-purple-500/60 rounded-2xl p-3 space-y-1 active:scale-95 transition';
      card.innerHTML = `<div class="text-2xl">${esc(d.icon)}</div><div class="text-xs font-bold text-white">${esc(d.name)}</div><div class="text-[11px] text-slate-400 line-clamp-2">${esc(d.tagline)}</div>`;
      card.onclick = () => {
        triggerHaptic('light');
        renderDomain(d.id);
      };
      dom.pickerGrid.appendChild(card);
    });
    showPicker();
  }

  function showPicker() {
    if (recording) return; // finish the voice turn first
    state.sessionId = null; // replies still in flight belong to the topic being left
    dom.picker.classList.remove('hidden');
    dom.view.classList.add('hidden');
    dom.topBar.classList.add('hidden');
    dom.bottomBar.classList.add('hidden');
    window.scrollTo(0, 0);
  }

  async function renderDomain(domainId) {
    const d = state.domains[domainId];
    if (!d) return;

    state.domainId = domainId;
    state.sessionId = null;
    state.cart = { items: [], count: 0, total_uah: 0, total_stars: 0 };
    updateCartUI();
    try {
      const s = await api.json('POST', '/v1/sessions', { topic: domainId });
      state.sessionId = s.session_id;
    } catch (err) {
      dom.conciergeMessage.textContent = 'Не вдалося почати сесію: ' + err.message;
    }
    dom.domainName.textContent = d.name;
    dom.picker.classList.add('hidden');
    dom.view.classList.remove('hidden');
    dom.topBar.classList.remove('hidden');
    dom.bottomBar.classList.remove('hidden');
    window.scrollTo(0, 0);
    dom.domainIcon.textContent = d.icon;
    dom.domainTagline.textContent = d.tagline;
    dom.conciergeMessage.textContent = d.welcome;
    dom.itemsCountBadge.textContent = (d.items || []).length ? `${d.items.length} ${plural(d.items.length, 'позиція', 'позиції', 'позицій')}` : 'лише розмова'; // topics without a catalog have nothing to order

    // Швидкі підказки
    dom.quickPrompts.innerHTML = '';
    (d.quick_prompts || []).forEach(promptText => {
      const btn = document.createElement('button');
      btn.className = 'bg-slate-800/90 hover:bg-slate-700/90 border border-slate-700/70 text-slate-300 text-xs px-3 py-1.5 rounded-full whitespace-nowrap active:scale-95 transition shadow-sm';
      btn.textContent = promptText;
      btn.onclick = () => {
        triggerHaptic('light');
        sendQuery(promptText);
      };
      dom.quickPrompts.appendChild(btn);
    });

    // Каталог товарів
    dom.catalogItems.innerHTML = '';
    (d.items || []).forEach(item => {
      const card = document.createElement('div');
      card.id = `item-card-${item.item_id}`;
      card.className = 'item-card bg-slate-900/90 border border-slate-800 rounded-2xl p-3.5 flex items-center justify-between gap-3 transition-all duration-300 shadow-sm';
      card.innerHTML = `
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-12 h-12 rounded-xl bg-slate-800 flex items-center justify-center text-2xl shrink-0 border border-slate-700/40">
            ${esc(item.icon || '📦')}
          </div>
          <div class="min-w-0">
            <div class="flex items-center gap-1.5">
              <h4 class="text-xs font-bold text-white truncate">${esc(item.title)}</h4>
              ${item.badge ? `<span class="bg-purple-950 text-purple-300 border border-purple-500/30 text-[9px] font-bold px-1.5 py-0.2 rounded-full">${esc(item.badge)}</span>` : ''}
            </div>
            <p class="text-[11px] text-slate-400 line-clamp-1 mt-0.5">${esc(item.description)}</p>
            <div class="flex items-center gap-2 mt-1">
              <span class="text-xs font-black text-slate-200">${item.price_uah} грн</span>
              <span class="text-[11px] font-bold text-amber-400 bg-amber-400/10 px-1.5 py-0.5 rounded flex items-center gap-0.5">
                <span>⭐</span><span>${item.price_stars}</span>
              </span>
            </div>
          </div>
        </div>
        <button class="add-btn shrink-0 w-8 h-8 rounded-xl bg-slate-800 hover:bg-purple-600 text-purple-300 hover:text-white border border-slate-700 flex items-center justify-center font-bold text-base transition active:scale-90" data-id="${esc(item.item_id)}">
          +
        </button>
      `;

      card.querySelector('.add-btn').onclick = () => {
        triggerHaptic('medium');
        addToCart(item.item_id);
      };

      dom.catalogItems.appendChild(card);
    });
  }

  const qtyOf = itemId => qtyIn(state.cart, itemId);

  // The server owns the cart: every change is a PUT and the response is the new truth.
  async function setQty(itemId, quantity) {
    if (!state.sessionId) return;
    try {
      state.cart = await api.json('PUT', `/v1/sessions/${state.sessionId}/cart/items/${encodeURIComponent(itemId)}`, { quantity });
      updateCartUI();
    } catch (err) {
      notify('Не вдалося змінити кошик: ' + err.message);
    }
  }

  const addToCart = itemId => setQty(itemId, qtyOf(itemId) + 1);
  const removeFromCart = itemId => setQty(itemId, 0);

  function updateCartUI() {
    const { count: totalCount, total_uah: totalUAH, total_stars: totalStars } = state.cart;

    if (totalCount > 0) {
      dom.cartBar.classList.remove('hidden');
      dom.cartBar.classList.add('flex');
      dom.cartSummaryText.textContent = `${totalCount} ${plural(totalCount, 'позиція', 'позиції', 'позицій')}`;
      dom.cartSubtotalText.textContent = `${totalUAH} грн • ${totalStars} ⭐ Stars`;
      dom.checkoutStarsBadge.textContent = `${totalStars} ⭐`;
    } else {
      dom.cartBar.classList.add('hidden');
      dom.cartBar.classList.remove('flex');
    }

    // Оновлення модалки
    dom.modalTotalUAH.textContent = `${totalUAH} грн`;
    dom.modalTotalStars.textContent = `${totalStars} ⭐`;
    renderModalItems();
  }

  function renderModalItems() {
    dom.modalCartItems.innerHTML = '';
    if (state.cart.items.length === 0) {
      dom.modalCartItems.innerHTML = '<div class="text-xs text-slate-500 text-center py-4">Кошик порожній</div>';
      return;
    }

    state.cart.items.forEach(item => {
      const row = document.createElement('div');
      row.className = 'flex items-center justify-between bg-slate-800/60 p-2.5 rounded-xl border border-slate-700/50';
      row.innerHTML = `
        <div class="flex items-center gap-2 min-w-0">
          <span class="text-lg">${esc(item.icon)}</span>
          <div class="min-w-0">
            <div class="text-xs font-bold text-white truncate">${esc(item.title)}</div>
            <div class="text-[10px] text-slate-400">${item.quantity} x ${item.price_uah} грн (${item.price_stars} ⭐)</div>
          </div>
        </div>
        <button class="remove-btn text-rose-400 hover:text-rose-300 p-1 text-xs">✕</button>
      `;
      row.querySelector('.remove-btn').onclick = () => {
        triggerHaptic('light');
        removeFromCart(item.item_id);
      };
      dom.modalCartItems.appendChild(row);
    });
  }

  // A reply, from text or voice: the cart is the server's copy, ui_actions only animate.
  function showReply(data, heard) {
    dom.conciergeMessage.textContent = (heard ? `🎤 «${heard}»\n\n` : '') + data.reply;
    if (state.voiceMode) {
      if (data.audio_b64) playAudio(data.audio_b64, data.audio_type); // the engine's voice
      else if (window.speechSynthesis) speakText(data.reply); // free browser fallback
    }
    state.cart = data.cart;
    updateCartUI();
    applyUIActions(data.ui_actions || []);
  }

  // 7. Відправка запитів до AI-Консьєржа
  async function sendQuery(text) {
    if (!text || !text.trim()) return;
    if (!state.sessionId) {
      dom.conciergeMessage.textContent = 'Сесія ще не почалась. Оберіть тему ще раз або спробуйте пізніше.';
      return;
    }
    const sid = state.sessionId;

    dom.conciergeMessage.textContent = 'Обмірковую відповідь...';
    animateWave(true);

    try {
      const data = await api.json('POST', `/v1/sessions/${sid}/messages`, { text, lang: LANG, speak: state.voiceMode });
      animateWave(false);
      if (sid === state.sessionId) showReply(data); // else: the topic changed meanwhile
    } catch (err) {
      animateWave(false);
      if (sid === state.sessionId) dom.conciergeMessage.textContent = 'Перепрошую, сталася помилка зʼєднання. Спробуйте ще раз.';
    }
  }

  // 8. Виконання UI-дій, отриманих від LLM
  function applyUIActions(actions) {
    actions.forEach(act => {
      switch (act.type) {
        case 'HIGHLIGHT_ITEM': {
          const card = document.getElementById(`item-card-${act.payload.item_id}`);
          if (card) {
            triggerHaptic('medium');
            card.scrollIntoView({ behavior: 'smooth', block: 'center' });
            card.classList.add('ring-2', 'ring-purple-500', 'bg-purple-950/40');
            setTimeout(() => {
              card.classList.remove('ring-2', 'ring-purple-500', 'bg-purple-950/40');
            }, act.payload.duration_ms || 2500);
          }
          break;
        }

        case 'ADD_TO_CART': {
          triggerHaptic('success'); // the cart itself already came back in the response
          break;
        }

        case 'OPEN_CHECKOUT': {
          triggerHaptic('medium');
          openCheckoutModal();
          break;
        }
      }
    });
  }

  // 9. Озвучення тексту (Client-Side Zero-Cost SpeechSynthesis)
  function speakText(text) {
    if (!window.speechSynthesis) return;
    window.speechSynthesis.cancel();
    const utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = 'uk-UA';
    utterance.rate = 1.05;
    animateWave(true);
    utterance.onend = () => animateWave(false);
    utterance.onerror = () => animateWave(false);
    window.speechSynthesis.speak(utterance);
  }

  // 10. Анімація звукової хвилі
  let waveInterval = null;
  function animateWave(active) {
    if (active) {
      if (waveInterval) clearInterval(waveInterval);
      waveInterval = setInterval(() => {
        dom.waveBars.forEach(bar => {
          const h = Math.floor(Math.random() * 20) + 4;
          bar.style.height = `${h}px`;
        });
      }, 90);
    } else {
      if (waveInterval) clearInterval(waveInterval);
      dom.waveBars.forEach((bar, idx) => {
        bar.style.height = `${[6, 8, 12, 6, 10, 8, 5][idx % 7]}px`;
      });
    }
  }

  // 11. Голос. Основний шлях: запис мікрофона -> STT рушія (працює у вебв'ю Telegram,
  // де Web Speech часто немає). Без MediaRecorder -- запасний браузерний розпізнавач.
  let recording = null;

  function setMicBusy(on) {
    dom.micBtn.classList.toggle('pulse-record', on);
    dom.micBtn.classList.toggle('bg-rose-600', on);
    dom.micBtn.classList.toggle('bg-purple-600', !on);
    animateWave(on);
  }

  let voiceBusy = false; // opening the mic, recording or uploading: one voice turn at a time

  async function toggleMic() {
    if (recording) { recording.stop(); return; }
    if (!recorder.recorderSupported()) { listenWithBrowser(); return; }
    if (!state.sessionId || voiceBusy) return;
    voiceBusy = true;
    const sid = state.sessionId;
    try {
      recording = await recorder.start();
    } catch (err) {
      voiceBusy = false;
      notify(err.name === 'NotAllowedError' ? 'Немає доступу до мікрофона. Дозвольте його або пишіть текстом.' : 'Запис недоступний: ' + err.message);
      return;
    }
    triggerHaptic('medium');
    setMicBusy(true);
    dom.conciergeMessage.textContent = 'Слухаю вас... Натисніть ще раз, щоб надіслати';
    let rec;
    try {
      rec = await recording.done;
    } catch (err) {
      recording = null; voiceBusy = false; setMicBusy(false);
      dom.conciergeMessage.textContent = 'Запис перервано: ' + err.message;
      return;
    }
    recording = null;
    dom.conciergeMessage.textContent = 'Розпізнаю...';
    try {
      const data = await api.voice(sid, rec.blob, rec.seconds, state.voiceMode);
      if (sid !== state.sessionId) return; // the topic changed meanwhile: that answer is for another conversation
      showReply(data, data.transcript);
    } catch (err) {
      if (sid === state.sessionId) dom.conciergeMessage.textContent = 'Не вдалося обробити голос: ' + err.message;
    } finally {
      voiceBusy = false;
      setMicBusy(false);
    }
  }

  let playing = null;

  function playAudio(b64, type) {
    if (playing) playing(); // never two clips on top of each other
    const url = URL.createObjectURL(new Blob([b64ToBytes(b64)], { type: type || 'audio/ogg' }));
    const a = new Audio(url);
    const free = () => {
      if (playing !== free) return;
      playing = null;
      a.pause();
      URL.revokeObjectURL(url);
      animateWave(false);
    };
    playing = free;
    a.onended = a.onerror = free;
    animateWave(true);
    a.play().catch(free); // the webview may refuse autoplay; the text is already shown
  }

  function listenWithBrowser() {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    if (!SpeechRecognition) {
      alert('Голосове розпізнавання не підтримується цим браузером. Використовуйте текстовий ввід.');
      return;
    }

    if (!state.isRecording) {
      const recognition = new SpeechRecognition();
      recognition.lang = 'uk-UA';
      recognition.interimResults = false;

      recognition.onstart = () => {
        state.isRecording = true;
        triggerHaptic('medium');
        dom.micBtn.classList.add('pulse-record', 'bg-rose-600');
        dom.micBtn.classList.remove('bg-purple-600');
        dom.conciergeMessage.textContent = 'Слухаю вас... Говоріть';
        animateWave(true);
      };

      recognition.onresult = (e) => {
        const transcript = e.results[0][0].transcript;
        dom.textInput.value = transcript;
        sendQuery(transcript);
      };

      recognition.onerror = () => {
        state.isRecording = false;
        animateWave(false);
        dom.micBtn.classList.remove('pulse-record', 'bg-rose-600');
        dom.micBtn.classList.add('bg-purple-600');
      };

      recognition.onend = () => {
        state.isRecording = false;
        animateWave(false);
        dom.micBtn.classList.remove('pulse-record', 'bg-rose-600');
        dom.micBtn.classList.add('bg-purple-600');
      };

      recognition.start();
    }
  }

  // 12. Оплата через Telegram Stars
  async function clearCart() {
    try {
      state.cart = await api.json('DELETE', `/v1/sessions/${state.sessionId}/cart`);
    } catch (err) {
      console.warn('clear cart failed:', err);
    }
    updateCartUI();
    closeCheckoutModal();
  }

  async function payWithStars() {
    if (state.cart.count <= 0) return;

    triggerHaptic('medium');
    dom.payStarsBtn.disabled = true;
    dom.payStarsBtn.textContent = 'Формуємо рахунок у Stars...';
    const resetBtn = () => {
      dom.payStarsBtn.disabled = false;
      dom.payStarsBtn.innerHTML = '<span>Підтвердити оплату через Stars</span><span>⭐</span>';
    };

    try {
      // No amounts are sent: the server prices its own copy of the cart.
      const data = await api.json('POST', `/v1/sessions/${state.sessionId}/checkout`, {});
      resetBtn();

      if (!data.mock && tg?.openInvoice) {
        // Офіційна платіжна шторка Telegram Stars
        tg.openInvoice(data.invoice_url, (status) => {
          if (status === 'paid') {
            triggerHaptic('success');
            notify('Оплата через Telegram Stars успішна! Дякуємо за замовлення.');
            clearCart();
          } else {
            triggerHaptic('warning');
          }
        });
      } else {
        // Демо-режим: сервер без токена бота повертає фейкове посилання
        triggerHaptic('success');
        notify(`[Demo] Рахунок на ${data.stars} ⭐ сформовано (оплата не виконується).`);
        clearCart();
      }
    } catch (err) {
      resetBtn();
      notify('Помилка формування рахунку: ' + err.message);
    }
  }

  function openCheckoutModal() {
    dom.checkoutModal.classList.remove('hidden');
  }

  function closeCheckoutModal() {
    dom.checkoutModal.classList.add('hidden');
  }

  // 13. Обробники подій
  dom.topicBtn.onclick = () => {
    triggerHaptic('light');
    showPicker();
  };

  dom.voiceToggle.onclick = () => {
    triggerHaptic('light');
    state.voiceMode = !state.voiceMode;
    dom.voiceLabel.textContent = state.voiceMode ? '🔊 З озвучкою' : '🔇 Без озвучки';
    dom.voiceLabel.className = state.voiceMode ? 'text-purple-400 font-bold' : 'text-slate-400';
  };

  dom.sendBtn.onclick = () => {
    const val = dom.textInput.value;
    dom.textInput.value = '';
    sendQuery(val);
  };

  dom.textInput.onkeydown = (e) => {
    if (e.key === 'Enter') {
      const val = dom.textInput.value;
      dom.textInput.value = '';
      sendQuery(val);
    }
  };

  dom.micBtn.onclick = toggleMic;

  dom.checkoutBtn.onclick = () => {
    triggerHaptic('medium');
    openCheckoutModal();
  };

  dom.closeModalBtn.onclick = () => {
    closeCheckoutModal();
  };

  dom.payStarsBtn.onclick = () => {
    payWithStars();
  };

  // Початковий старт
  loadDomains();
