/**
 * Enhanced Material Design 3 Interactions
 * Advanced animations and micro-interactions for professional UI
 */

class MD3Interactions {
  constructor() {
    this.init();
  }

  init() {
    this.setupRippleEffects();
    this.setupStateLayerAnimations();
    this.setupModalAnimations();
    this.setupScrollAnimations();
    this.setupToastNotifications();
    this.setupFormValidation();
    this.setupChartInteractions();
    this.setupNavigationAnimations();
    this.setupLoadingStates();
    this.setupAccessibilityFeatures();
  }

  // Material Design 3 Ripple Effect
  setupRippleEffects() {
    const rippleElements = document.querySelectorAll('.md3-button, .md3-icon-button, .md3-fab, .md3-card.md3-state-layer');
    
    rippleElements.forEach(element => {
      element.addEventListener('click', (e) => {
        this.createRipple(e, element);
      });
    });
  }

  createRipple(event, element) {
    const ripple = document.createElement('span');
    const rect = element.getBoundingClientRect();
    const size = Math.max(rect.width, rect.height);
    const x = event.clientX - rect.left - size / 2;
    const y = event.clientY - rect.top - size / 2;
    
    ripple.style.cssText = `
      position: absolute;
      width: ${size}px;
      height: ${size}px;
      left: ${x}px;
      top: ${y}px;
      background: currentColor;
      border-radius: 50%;
      opacity: 0.12;
      transform: scale(0);
      pointer-events: none;
      z-index: 10;
    `;
    
    const existingRipple = element.querySelector('.md3-ripple');
    if (existingRipple) {
      existingRipple.remove();
    }
    
    ripple.className = 'md3-ripple';
    element.style.position = 'relative';
    element.style.overflow = 'hidden';
    element.appendChild(ripple);
    
    // Animate ripple
    ripple.animate([
      { transform: 'scale(0)', opacity: 0.12 },
      { transform: 'scale(1)', opacity: 0 }
    ], {
      duration: 300,
      easing: 'cubic-bezier(0.2, 0, 0, 1)'
    }).onfinish = () => ripple.remove();
  }

  // Enhanced State Layer Animations
  setupStateLayerAnimations() {
    const stateElements = document.querySelectorAll('.md3-state-layer');
    
    stateElements.forEach(element => {
      element.addEventListener('mouseenter', () => {
        this.animateStateLayer(element, 'hover', true);
      });
      
      element.addEventListener('mouseleave', () => {
        this.animateStateLayer(element, 'hover', false);
      });
      
      element.addEventListener('focus', () => {
        this.animateStateLayer(element, 'focus', true);
      });
      
      element.addEventListener('blur', () => {
        this.animateStateLayer(element, 'focus', false);
      });
    });
  }

  animateStateLayer(element, state, isActive) {
    let stateLayer = element.querySelector('.md3-state-layer-overlay');
    
    if (!stateLayer) {
      stateLayer = document.createElement('div');
      stateLayer.className = 'md3-state-layer-overlay';
      stateLayer.style.cssText = `
        position: absolute;
        inset: 0;
        background: currentColor;
        opacity: 0;
        pointer-events: none;
        border-radius: inherit;
        z-index: 1;
      `;
      element.style.position = 'relative';
      element.appendChild(stateLayer);
    }
    
    const targetOpacity = isActive ? (state === 'hover' ? 0.08 : 0.12) : 0;
    
    stateLayer.animate([
      { opacity: stateLayer.style.opacity || 0 },
      { opacity: targetOpacity }
    ], {
      duration: 150,
      easing: 'cubic-bezier(0.2, 0, 0, 1)',
      fill: 'forwards'
    });
  }

  // Modal Animations
  setupModalAnimations() {
    const modals = document.querySelectorAll('[id$="-modal"]');
    
    modals.forEach(modal => {
      const dialog = modal.querySelector('[id$="-dialog"]');
      
      modal.addEventListener('click', (e) => {
        if (e.target === modal) {
          this.hideModal(modal);
        }
      });
    });
  }

  showModal(modalId) {
    const modal = document.getElementById(modalId);
    const dialog = modal.querySelector('[id$="-dialog"]');
    
    if (!modal || !dialog) return;
    
    modal.classList.remove('hidden');
    
    // Animate backdrop
    modal.animate([
      { opacity: 0 },
      { opacity: 1 }
    ], {
      duration: 200,
      easing: 'cubic-bezier(0.2, 0, 0, 1)',
      fill: 'forwards'
    });
    
    // Animate dialog
    dialog.animate([
      { transform: 'scale(0.9)', opacity: 0 },
      { transform: 'scale(1)', opacity: 1 }
    ], {
      duration: 250,
      easing: 'cubic-bezier(0.05, 0.7, 0.1, 1)',
      fill: 'forwards'
    });
    
    // Focus management
    const focusableElements = dialog.querySelectorAll('button, input, select, textarea, [tabindex]:not([tabindex="-1"])');
    if (focusableElements.length > 0) {
      focusableElements[0].focus();
    }
  }

  hideModal(modal) {
    const dialog = modal.querySelector('[id$="-dialog"]');
    
    // Animate out
    const hideAnimation = dialog.animate([
      { transform: 'scale(1)', opacity: 1 },
      { transform: 'scale(0.9)', opacity: 0 }
    ], {
      duration: 200,
      easing: 'cubic-bezier(0.4, 0, 1, 1)',
      fill: 'forwards'
    });
    
    modal.animate([
      { opacity: 1 },
      { opacity: 0 }
    ], {
      duration: 200,
      easing: 'cubic-bezier(0.4, 0, 1, 1)',
      fill: 'forwards'
    });
    
    hideAnimation.onfinish = () => {
      modal.classList.add('hidden');
    };
  }

  // Scroll-based Animations
  setupScrollAnimations() {
    const observerOptions = {
      threshold: 0.1,
      rootMargin: '0px 0px -50px 0px'
    };
    
    const observer = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          this.animateIntoView(entry.target);
        }
      });
    }, observerOptions);
    
    // Observe cards and important elements with staggered animation support
    document.querySelectorAll('.md3-card, .metric-card, .agent-card').forEach((el, index) => {
      el.style.setProperty('--stagger-index', index);
      observer.observe(el);
    });
    
    // Setup parallax effect for hero sections
    this.setupParallaxEffects();
  }

  animateIntoView(element) {
    if (element.dataset.animated === 'true') return;
    
    element.dataset.animated = 'true';
    element.classList.add('animate-fadeIn');
    
    // Add hover effects after animation
    setTimeout(() => {
      element.classList.add('hover-lift');
    }, 300);
  }

  setupParallaxEffects() {
    const parallaxElements = document.querySelectorAll('[data-parallax]');
    
    if (parallaxElements.length === 0) return;
    
    const handleScroll = this.throttle(() => {
      const scrollY = window.pageYOffset;
      
      parallaxElements.forEach(element => {
        const speed = parseFloat(element.dataset.parallax) || 0.5;
        const yPos = -(scrollY * speed);
        element.style.transform = `translateY(${yPos}px)`;
      });
    }, 16); // 60fps
    
    window.addEventListener('scroll', handleScroll, { passive: true });
  }

  // Toast Notifications
  setupToastNotifications() {
    this.toastContainer = this.createToastContainer();
  }

  createToastContainer() {
    let container = document.getElementById('toast-container');
    if (!container) {
      container = document.createElement('div');
      container.id = 'toast-container';
      container.style.cssText = `
        position: fixed;
        top: 24px;
        right: 24px;
        z-index: 1000;
        display: flex;
        flex-direction: column;
        gap: 12px;
        pointer-events: none;
      `;
      document.body.appendChild(container);
    }
    return container;
  }

  showToast(message, type = 'info', duration = 4000) {
    const toast = document.createElement('div');
    const id = `toast-${Date.now()}`;
    
    const icons = {
      success: 'check_circle',
      error: 'error',
      warning: 'warning',
      info: 'info'
    };
    
    toast.innerHTML = `
      <div class="flex items-center gap-3 p-4 bg-surface-container rounded-xl shadow-lg border border-outline-variant">
        <span class="material-symbols-outlined text-${type === 'error' ? 'error' : 'primary'}">${icons[type]}</span>
        <span class="text-on-surface">${message}</span>
        <button class="ml-auto p-1 hover:bg-surface-variant rounded-full transition-colors" onclick="md3.hideToast('${id}')">
          <span class="material-symbols-outlined text-sm">close</span>
        </button>
      </div>
    `;
    
    toast.id = id;
    toast.style.cssText = `
      transform: translateX(100%);
      opacity: 0;
      pointer-events: auto;
    `;
    
    this.toastContainer.appendChild(toast);
    
    // Animate in
    toast.animate([
      { transform: 'translateX(100%)', opacity: 0 },
      { transform: 'translateX(0)', opacity: 1 }
    ], {
      duration: 300,
      easing: 'cubic-bezier(0.05, 0.7, 0.1, 1)',
      fill: 'forwards'
    });
    
    // Auto-hide after duration
    if (duration > 0) {
      setTimeout(() => this.hideToast(id), duration);
    }
    
    return id;
  }

  hideToast(id) {
    const toast = document.getElementById(id);
    if (!toast) return;
    
    const hideAnimation = toast.animate([
      { transform: 'translateX(0)', opacity: 1 },
      { transform: 'translateX(100%)', opacity: 0 }
    ], {
      duration: 250,
      easing: 'cubic-bezier(0.4, 0, 1, 1)',
      fill: 'forwards'
    });
    
    hideAnimation.onfinish = () => toast.remove();
  }

  // Enhanced Form Validation
  setupFormValidation() {
    const forms = document.querySelectorAll('form');
    
    forms.forEach(form => {
      const inputs = form.querySelectorAll('input, textarea, select');
      
      inputs.forEach(input => {
        input.addEventListener('blur', () => {
          this.validateField(input);
        });
        
        input.addEventListener('input', () => {
          if (input.classList.contains('error')) {
            this.validateField(input);
          }
        });
      });
    });
  }

  validateField(field) {
    const value = field.value.trim();
    const isRequired = field.hasAttribute('required');
    const type = field.type;
    
    let isValid = true;
    let errorMessage = '';
    
    if (isRequired && !value) {
      isValid = false;
      errorMessage = 'This field is required';
    } else if (value && type === 'email' && !this.isValidEmail(value)) {
      isValid = false;
      errorMessage = 'Please enter a valid email address';
    }
    
    this.updateFieldValidation(field, isValid, errorMessage);
  }

  updateFieldValidation(field, isValid, errorMessage) {
    const container = field.closest('.space-y-2') || field.parentElement;
    let errorElement = container.querySelector('.field-error');
    
    if (isValid) {
      field.classList.remove('error');
      if (errorElement) errorElement.remove();
    } else {
      field.classList.add('error');
      
      if (!errorElement) {
        errorElement = document.createElement('p');
        errorElement.className = 'field-error text-error text-sm flex items-center gap-1';
        errorElement.innerHTML = `
          <span class="material-symbols-outlined text-sm">error</span>
          <span>${errorMessage}</span>
        `;
        container.appendChild(errorElement);
      } else {
        errorElement.querySelector('span:last-child').textContent = errorMessage;
      }
    }
  }

  isValidEmail(email) {
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  }

  // Chart Interactions
  setupChartInteractions() {
    const charts = document.querySelectorAll('.interactive-chart');
    
    charts.forEach(chart => {
      const bars = chart.querySelectorAll('.chart-bar');
      
      bars.forEach((bar, index) => {
        bar.addEventListener('mouseenter', () => {
          this.highlightChartData(bar, true);
        });
        
        bar.addEventListener('mouseleave', () => {
          this.highlightChartData(bar, false);
        });
      });
    });
  }

  highlightChartData(bar, isHighlighted) {
    if (isHighlighted) {
      bar.style.filter = 'brightness(1.1)';
      bar.style.transform = 'scaleY(1.05)';
    } else {
      bar.style.filter = 'none';
      bar.style.transform = 'scaleY(1)';
    }
  }

  // Navigation Animations
  setupNavigationAnimations() {
    const navItems = document.querySelectorAll('.nav-item');
    
    navItems.forEach(item => {
      item.addEventListener('click', () => {
        this.animateNavigation(item);
      });
    });
  }

  animateNavigation(activeItem) {
    // Remove active state from all items
    document.querySelectorAll('.nav-item').forEach(item => {
      item.classList.remove('active');
    });
    
    // Add active state to clicked item
    activeItem.classList.add('active');
    
    // Animate active indicator
    const indicator = document.querySelector('.nav-indicator');
    if (indicator) {
      const itemRect = activeItem.getBoundingClientRect();
      const railRect = activeItem.closest('.md3-navigation-rail').getBoundingClientRect();
      
      indicator.animate([
        { transform: `translateY(${indicator.offsetTop}px)` },
        { transform: `translateY(${itemRect.top - railRect.top + itemRect.height / 2 - indicator.offsetHeight / 2}px)` }
      ], {
        duration: 250,
        easing: 'cubic-bezier(0.2, 0, 0, 1)',
        fill: 'forwards'
      });
    }
  }

  // Loading States
  setupLoadingStates() {
    const loadingElements = document.querySelectorAll('[data-loading]');
    
    loadingElements.forEach(element => {
      this.observeLoadingState(element);
    });
  }

  observeLoadingState(element) {
    const originalContent = element.innerHTML;
    
    // Watch for htmx loading states
    element.addEventListener('htmx:beforeRequest', () => {
      this.showLoadingState(element, originalContent);
    });
    
    element.addEventListener('htmx:afterRequest', () => {
      this.hideLoadingState(element, originalContent);
    });
  }

  showLoadingState(element, originalContent) {
    element.setAttribute('data-original-content', originalContent);
    element.innerHTML = `
      <div class="flex items-center justify-center gap-3 p-4">
        <div class="animate-spin w-5 h-5 border-2 border-primary border-t-transparent rounded-full"></div>
        <span class="text-on-surface-variant">Loading...</span>
      </div>
    `;
  }

  hideLoadingState(element, originalContent) {
    const savedContent = element.getAttribute('data-original-content');
    if (savedContent) {
      element.innerHTML = savedContent;
      element.removeAttribute('data-original-content');
    }
  }

  // Accessibility Features
  setupAccessibilityFeatures() {
    this.setupKeyboardNavigation();
    this.setupFocusManagement();
    this.setupScreenReaderSupport();
  }

  setupKeyboardNavigation() {
    document.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        // Close modals
        const openModal = document.querySelector('[id$="-modal"]:not(.hidden)');
        if (openModal) {
          this.hideModal(openModal);
        }
        
        // Close dropdowns
        const openDropdown = document.querySelector('.dropdown.open');
        if (openDropdown) {
          openDropdown.classList.remove('open');
        }
      }
    });
  }

  setupFocusManagement() {
    const focusableElements = document.querySelectorAll('button, input, select, textarea, a[href], [tabindex]:not([tabindex="-1"])');
    
    focusableElements.forEach(element => {
      element.addEventListener('focus', () => {
        element.classList.add('focused');
      });
      
      element.addEventListener('blur', () => {
        element.classList.remove('focused');
      });
    });
  }

  setupScreenReaderSupport() {
    // Announce dynamic content changes
    const liveRegion = document.createElement('div');
    liveRegion.setAttribute('aria-live', 'polite');
    liveRegion.setAttribute('aria-atomic', 'true');
    liveRegion.className = 'sr-only';
    liveRegion.id = 'live-region';
    document.body.appendChild(liveRegion);
    
    this.liveRegion = liveRegion;
  }

  announceToScreenReader(message) {
    if (this.liveRegion) {
      this.liveRegion.textContent = message;
      setTimeout(() => {
        this.liveRegion.textContent = '';
      }, 1000);
    }
  }

  // Utility Methods
  debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }

  throttle(func, limit) {
    let inThrottle;
    return function() {
      const args = arguments;
      const context = this;
      if (!inThrottle) {
        func.apply(context, args);
        inThrottle = true;
        setTimeout(() => inThrottle = false, limit);
      }
    };
  }
}

// Global Functions for Template Integration
window.showModal = function(modalId) {
  window.md3.showModal(modalId);
};

window.hideModal = function(modalId) {
  const modal = document.getElementById(modalId);
  if (modal) window.md3.hideModal(modal);
};

window.showToast = function(message, type = 'info', duration = 4000) {
  return window.md3.showToast(message, type, duration);
};

window.hideToast = function(id) {
  window.md3.hideToast(id);
};

// Table interaction functions
window.toggleAllRows = function(checkbox) {
  const table = checkbox.closest('table');
  const rowCheckboxes = table.querySelectorAll('tbody input[type="checkbox"]');
  
  rowCheckboxes.forEach(cb => {
    cb.checked = checkbox.checked;
  });
  
  window.md3.announceToScreenReader(
    checkbox.checked ? 'All rows selected' : 'All rows deselected'
  );
};

window.sortTable = function(column) {
  window.md3.announceToScreenReader(`Sorting by ${column}`);
  // Implementation would depend on backend sorting
};

window.navigatePage = function(page) {
  window.md3.announceToScreenReader(`Navigating to page ${page}`);
  // Implementation would depend on backend pagination
};

window.selectRow = function(rowId) {
  window.md3.announceToScreenReader(`Row ${rowId} selected`);
};

window.editRow = function(rowId) {
  window.md3.announceToScreenReader(`Editing row ${rowId}`);
};

window.deleteRow = function(rowId) {
  if (confirm('Are you sure you want to delete this item?')) {
    window.md3.announceToScreenReader(`Row ${rowId} deleted`);
  }
};

window.showRowMenu = function(rowId) {
  window.md3.announceToScreenReader(`Showing menu for row ${rowId}`);
};

window.closeModal = function(event) {
  const modal = event.target.closest('[id$="-modal"]');
  if (modal && window.md3) {
    window.md3.hideModal(modal);
  }
};

window.updateFileLabel = function(input) {
  const label = document.getElementById(input.name + '-label');
  if (label && input.files.length > 0) {
    label.textContent = input.files[0].name;
  }
};

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  window.md3 = new MD3Interactions();
  console.log('Material Design 3 Enhanced Interactions initialized');
});

// Enhanced Chat-specific Functions
window.clearChat = function() {
  if (confirm('Are you sure you want to clear the chat?')) {
    const container = document.getElementById('messages-container');
    container.innerHTML = '';
    // Reset to empty state
    container.innerHTML = `
      <div class="h-full flex items-center justify-center">
        <div class="text-center max-w-2xl">
          <div class="w-16 h-16 bg-gradient-to-br from-primary via-secondary to-tertiary rounded-2xl flex items-center justify-center mx-auto mb-6 shadow-lg">
            <span class="material-symbols-outlined text-white text-2xl">auto_awesome</span>
          </div>
          <h2 class="text-2xl font-semibold text-on-surface mb-3">開始與AI助手對話</h2>
          <p class="text-on-surface-variant mb-8 text-lg">選擇一個話題開始，或者直接提出您的問題</p>
        </div>
      </div>
    `;
    if (window.md3) {
      window.md3.showToast('Chat cleared', 'info');
    }
  }
};

window.setPrompt = function(prompt) {
  const input = document.getElementById('message-input');
  if (input) {
    input.value = prompt;
    input.focus();
    autoResize(input);
    updateSendButton();
  }
};

window.selectAgent = function(agentId) {
  // Implement agent selection logic
  console.log('Selected agent:', agentId);
  if (window.md3) {
    window.md3.showToast('Agent selected', 'success');
  }
};

window.autoResize = function(textarea) {
  textarea.style.height = 'auto';
  const newHeight = Math.min(textarea.scrollHeight, 128); // max-h-32 = 128px
  textarea.style.height = newHeight + 'px';
};

window.updateSendButton = function() {
  const input = document.getElementById('message-input');
  const button = document.getElementById('send-button');
  const charCount = document.getElementById('char-count');
  
  if (input && button) {
    const hasContent = input.value.trim().length > 0;
    button.disabled = !hasContent;
    button.classList.toggle('opacity-50', !hasContent);
  }
  
  if (input && charCount) {
    charCount.textContent = input.value.length;
    const isOverLimit = input.value.length > 2000;
    charCount.classList.toggle('text-error', isOverLimit);
  }
};

window.handleKeyDown = function(event) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault();
    const form = document.getElementById('chat-form');
    if (form && !document.getElementById('send-button').disabled) {
      form.requestSubmit();
    }
  }
};

window.clearInput = function() {
  const input = document.getElementById('message-input');
  if (input) {
    input.value = '';
    autoResize(input);
    updateSendButton();
  }
};

window.toggleVoiceInput = function() {
  // Placeholder for voice input functionality
  if (window.md3) {
    window.md3.showToast('Voice input not yet implemented', 'info');
  }
};

// Auto-scroll to bottom of messages
window.scrollToBottom = function() {
  const container = document.getElementById('messages-container');
  if (container) {
    container.scrollTop = container.scrollHeight;
  }
};

// Handle dynamic content from HTMX
document.addEventListener('htmx:afterSwap', () => {
  // Re-initialize interactions for new content
  if (window.md3) {
    window.md3.setupRippleEffects();
    window.md3.setupStateLayerAnimations();
    window.md3.setupScrollAnimations();
  }
  
  // Auto-scroll to bottom after new messages
  scrollToBottom();
});