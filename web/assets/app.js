/**
 * v2v-tma — Client Application Engine
 * Handles Telegram WebApp SDK, Voice/Text Dual Mode, UI Action Mutations, Cart & Stars Payments
 */

(function () {
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
    domainId: 'pizzeria',
    domains: {},
    cart: [], // { item_id, title, price_uah, price_stars, icon, quantity }
    voiceMode: false, // false = Текст (Zero-Cost), true = Голос On-Demand
    isRecording: false,
    mediaRecorder: null,
    audioChunks: []
  };

  // 3. Елементи DOM
  const dom = {
    domainSelect: document.getElementById('domain-select'),
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

  // 5. Завантаження доменів та каталогу
  async function loadDomains() {
    try {
      const res = await fetch('/api/domains');
      const data = await res.json();
      data.forEach(d => state.domains[d.id] = d);
      renderDomain(state.domainId);
    } catch (err) {
      console.warn('API error, using local fallback:', err);
    }
  }

  function renderDomain(domainId) {
    const d = state.domains[domainId];
    if (!d) return;

    state.domainId = domainId;
    dom.domainSelect.value = domainId;
    dom.domainIcon.textContent = d.icon;
    dom.domainTagline.textContent = d.tagline;
    dom.conciergeMessage.textContent = d.welcome;
    dom.itemsCountBadge.textContent = `${d.items.length} позицій`;

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
      card.id = `item-card-${item.id}`;
      card.className = 'item-card bg-slate-900/90 border border-slate-800 rounded-2xl p-3.5 flex items-center justify-between gap-3 transition-all duration-300 shadow-sm';
      card.innerHTML = `
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-12 h-12 rounded-xl bg-slate-800 flex items-center justify-center text-2xl shrink-0 border border-slate-700/40">
            ${item.icon || '📦'}
          </div>
          <div class="min-w-0">
            <div class="flex items-center gap-1.5">
              <h4 class="text-xs font-bold text-white truncate">${item.title}</h4>
              ${item.badge ? `<span class="bg-purple-950 text-purple-300 border border-purple-500/30 text-[9px] font-bold px-1.5 py-0.2 rounded-full">${item.badge}</span>` : ''}
            </div>
            <p class="text-[11px] text-slate-400 line-clamp-1 mt-0.5">${item.description}</p>
            <div class="flex items-center gap-2 mt-1">
              <span class="text-xs font-black text-slate-200">${item.price_uah} грн</span>
              <span class="text-[11px] font-bold text-amber-400 bg-amber-400/10 px-1.5 py-0.5 rounded flex items-center gap-0.5">
                <span>⭐</span><span>${item.price_stars}</span>
              </span>
            </div>
          </div>
        </div>
        <button class="add-btn shrink-0 w-8 h-8 rounded-xl bg-slate-800 hover:bg-purple-600 text-purple-300 hover:text-white border border-slate-700 flex items-center justify-center font-bold text-base transition active:scale-90" data-id="${item.id}">
          +
        </button>
      `;

      card.querySelector('.add-btn').onclick = () => {
        triggerHaptic('medium');
        addToCart(item);
      };

      dom.catalogItems.appendChild(card);
    });
  }

  // 6. Робота з кошиком
  function addToCart(item, quantity = 1) {
    const existing = state.cart.find(c => c.item_id === item.id || c.item_id === item.item_id);
    if (existing) {
      existing.quantity += quantity;
    } else {
      state.cart.push({
        item_id: item.id || item.item_id,
        title: item.title,
        price_uah: item.price_uah,
        price_stars: item.price_stars,
        icon: item.icon || '📦',
        quantity: quantity
      });
    }
    updateCartUI();
  }

  function removeFromCart(itemId) {
    state.cart = state.cart.filter(c => c.item_id !== itemId);
    updateCartUI();
  }

  function updateCartUI() {
    const totalCount = state.cart.reduce((sum, i) => sum + i.quantity, 0);
    const totalUAH = state.cart.reduce((sum, i) => sum + (i.price_uah * i.quantity), 0);
    const totalStars = state.cart.reduce((sum, i) => sum + (i.price_stars * i.quantity), 0);

    if (totalCount > 0) {
      dom.cartBar.classList.remove('hidden');
      dom.cartBar.classList.add('flex');
      dom.cartSummaryText.textContent = `${totalCount} ${totalCount === 1 ? 'позиція' : 'позицій'}`;
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
    if (state.cart.length === 0) {
      dom.modalCartItems.innerHTML = '<div class="text-xs text-slate-500 text-center py-4">Кошик порожній</div>';
      return;
    }

    state.cart.forEach(item => {
      const row = document.createElement('div');
      row.className = 'flex items-center justify-between bg-slate-800/60 p-2.5 rounded-xl border border-slate-700/50';
      row.innerHTML = `
        <div class="flex items-center gap-2 min-w-0">
          <span class="text-lg">${item.icon}</span>
          <div class="min-w-0">
            <div class="text-xs font-bold text-white truncate">${item.title}</div>
            <div class="text-[10px] text-slate-400">${item.quantity} x ${item.price_uah} грн (${item.price_stars} ⭐)</div>
          </div>
        </div>
        <button class="remove-btn text-rose-400 hover:text-rose-300 p-1 text-xs" data-id="${item.item_id}">✕</button>
      `;
      row.querySelector('.remove-btn').onclick = () => {
        triggerHaptic('light');
        removeFromCart(item.item_id);
      };
      dom.modalCartItems.appendChild(row);
    });
  }

  // 7. Відправка запитів до AI-Консьєржа
  async function sendQuery(text) {
    if (!text || !text.trim()) return;

    dom.conciergeMessage.textContent = 'Обмірковую відповідь...';
    animateWave(true);

    try {
      const res = await fetch('/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          domain_id: state.domainId,
          text: text,
          lang: 'uk',
          init_data: tg?.initData || 'dev_demo_user'
        })
      });

      const data = await res.json();
      animateWave(false);

      // Виведення репліки
      dom.conciergeMessage.textContent = data.reply_text;

      // Якщо активний голосовий режим — озвучуємо нативним браузерним SpeechSynthesis (Zero-Cost)
      if (state.voiceMode && window.speechSynthesis) {
        speakText(data.reply_text);
      }

      // Виконання UI-мутацій
      applyUIActions(data.ui_actions || []);

    } catch (err) {
      animateWave(false);
      dom.conciergeMessage.textContent = 'Перепрошую, сталася помилка зʼєднання. Спробуйте ще раз.';
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
          triggerHaptic('success');
          addToCart(act.payload, act.payload.quantity || 1);
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

  // 11. Голосове розпізнавання (Push-to-Talk)
  function toggleSpeechRecognition() {
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
  async function payWithStars() {
    const totalStars = state.cart.reduce((sum, i) => sum + (i.price_stars * i.quantity), 0);
    if (totalStars <= 0) return;

    triggerHaptic('medium');
    dom.payStarsBtn.disabled = true;
    dom.payStarsBtn.textContent = 'Формуємо рахунок у Stars...';

    try {
      const res = await fetch('/api/checkout/stars', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          domain_id: state.domainId,
          stars_amount: totalStars,
          cart_items: state.cart,
          init_data: tg?.initData || 'dev_demo_user'
        })
      });

      const data = await res.json();
      dom.payStarsBtn.disabled = false;
      dom.payStarsBtn.innerHTML = '<span>Підтвердити оплату через Stars</span><span>⭐</span>';

      if (tg?.openInvoice && data.invoice_url) {
        // Відкриваємо офіційну платіжну шторку Telegram Stars
        tg.openInvoice(data.invoice_url, (status) => {
          if (status === 'paid') {
            triggerHaptic('success');
            alert('Оплата через Telegram Stars успішна! Дякуємо за замовлення.');
            state.cart = [];
            updateCartUI();
            closeCheckoutModal();
          } else {
            triggerHaptic('warning');
          }
        });
      } else {
        // Демо-повідомлення
        triggerHaptic('success');
        alert(`[Demo Mode] Рахунок на ${totalStars} ⭐ Stars згенеровано!\nInvoice: ${data.invoice_url}`);
        state.cart = [];
        updateCartUI();
        closeCheckoutModal();
      }
    } catch (err) {
      dom.payStarsBtn.disabled = false;
      dom.payStarsBtn.innerHTML = '<span>Підтвердити оплату через Stars</span><span>⭐</span>';
      alert('Помилка генерації рахунку Stars.');
    }
  }

  function openCheckoutModal() {
    dom.checkoutModal.classList.remove('hidden');
  }

  function closeCheckoutModal() {
    dom.checkoutModal.classList.add('hidden');
  }

  // 13. Обробники подій
  dom.domainSelect.onchange = (e) => {
    triggerHaptic('light');
    renderDomain(e.target.value);
  };

  dom.voiceToggle.onclick = () => {
    triggerHaptic('light');
    state.voiceMode = !state.voiceMode;
    dom.voiceLabel.textContent = state.voiceMode ? 'Голос On' : 'Текст';
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

  dom.micBtn.onclick = () => {
    toggleSpeechRecognition();
  };

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
})();
