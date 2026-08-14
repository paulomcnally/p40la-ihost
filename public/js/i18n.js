const I18n = (function () {
  let dictionary = {};
  let currentLang = localStorage.getItem('p40la-lang') || 'es';

  async function load(lang) {
    try {
      const res = await fetch(`/i18n/${lang}.json`);
      if (!res.ok) throw new Error('Failed to load dictionary');
      dictionary = await res.json();
      currentLang = lang;
      localStorage.setItem('p40la-lang', lang);
      document.documentElement.lang = lang;
      apply();
      return true;
    } catch (err) {
      console.error('i18n load error:', err);
      return false;
    }
  }

  function get(key, fallback) {
    const parts = key.split('.');
    let value = dictionary;
    for (const part of parts) {
      if (value == null || typeof value !== 'object') return fallback || key;
      value = value[part];
    }
    if (typeof value === 'string') return value;
    return fallback || key;
  }

  function apply() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
      const key = el.getAttribute('data-i18n');
      el.textContent = get(key);
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
      const key = el.getAttribute('data-i18n-placeholder');
      el.placeholder = get(key);
    });
  }

  function current() {
    return currentLang;
  }

  return {
    load,
    get,
    apply,
    current
  };
})();
