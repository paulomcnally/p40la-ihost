const App = (function () {
  const state = {
    homes: [],
    currencies: [],
    services: [],
    language: 'es'
  };

  const routes = {
    home: { title: 'home.title', render: renderHomePage },
    'home/new': { title: 'home.create', render: renderHomeFormPage },
    'home/edit': { title: 'home.edit', render: renderHomeFormPage },
    services: { title: 'services.title', render: renderServicesPage },
    'services/new': { title: 'services.create', render: renderServiceFormPage },
    'services/edit': { title: 'services.edit', render: renderServiceFormPage },
    'services/bills': { title: 'bills.title', render: renderBillsPage },
    'bills/new': { title: 'bills.create', render: renderBillFormPage },
    'bills/edit': { title: 'bills.edit', render: renderBillFormPage },
    settings: { title: 'settings.title', render: renderSettingsPage },
    'settings/language': { title: 'settings.language.title', render: renderLanguagePage },
    'settings/currency': { title: 'settings.currencies.create', render: renderCurrencyFormPage }
  };

  async function init() {
    const savedLang = localStorage.getItem('p40la-lang') || 'es';
    await I18n.load(savedLang);
    state.language = savedLang;

    document.body.innerHTML = `
      <aside class="sidebar" id="sidebar"></aside>
      <div class="main">
        <header class="header" id="header"></header>
        <main class="content" id="content"></main>
      </div>
    `;

    renderSidebar();
    renderHeader();

    window.addEventListener('popstate', () => handleRoute());
    handleRoute();
  }

  function getRoute() {
    const path = window.location.pathname.replace(/^\//, '') || 'home';
    const parts = path.split('/');
    if (parts[0] === '') parts[0] = 'home';

    const base = parts[0];
    const sub = parts[1];
    const id = parts[2];

    if (base === 'home' && sub === 'new') return { name: 'home/new', id: null };
    if (base === 'home' && sub === 'edit' && id) return { name: 'home/edit', id };
    if (base === 'services' && sub === 'new') return { name: 'services/new', id: null };
    if (base === 'services' && sub === 'edit' && id) return { name: 'services/edit', id };
    if (base === 'services' && sub === 'bills' && id) return { name: 'services/bills', id };
    if (base === 'bills' && sub === 'new') return { name: 'bills/new', id: null };
    if (base === 'bills' && sub === 'edit' && id) return { name: 'bills/edit', id };
    if (base === 'settings' && sub === 'currency') return { name: 'settings/currency', id };
    if (base === 'settings' && sub === 'language') return { name: 'settings/language', id: null };
    if (base === 'home') return { name: 'home', id: null };
    if (base === 'services') return { name: 'services', id: null };
    if (base === 'settings') return { name: 'settings', id: null };

    return { name: 'home', id: null };
  }

  async function handleRoute() {
    const route = getRoute();
    const config = routes[route.name] || routes.home;

    renderSidebar(route.name);
    setHeaderTitle(config.title);

    const content = document.getElementById('content');
    content.innerHTML = `<div class="empty-state">${I18n.get('app.loading')}</div>`;

    try {
      await config.render(route.id);
    } catch (err) {
      console.error('Route render error:', err);
      content.innerHTML = `<div class="empty-state"><p>${err.message || I18n.get('errors.generic')}</p></div>`;
    }
  }

  function navigate(path) {
    if (window.location.pathname === '/' + path) return;
    window.history.pushState({}, '', '/' + path);
    handleRoute();
  }

  async function loadData() {
    const [homes, currencies] = await Promise.all([
      API.homes.list(),
      API.currencies.list()
    ]);
    state.homes = homes || [];
    state.currencies = currencies || [];
  }

  async function refreshServices(homeId) {
    state.services = await API.services.list(homeId) || [];
  }

  function renderSidebar(activeRoute) {
    const activeBase = activeRoute ? activeRoute.split('/')[0] : 'home';
    const sidebar = document.getElementById('sidebar');
    sidebar.innerHTML = `
      <div class="sidebar-brand">${I18n.get('app.title')}</div>
      <nav class="sidebar-nav">
        <div class="sidebar-menu">
          <div class="sidebar-menu-title">${I18n.get('app.title')}</div>
          <a class="sidebar-item ${activeBase === 'home' ? 'active' : ''}" data-page="home">
            ${Icons.get('home')}
            <span data-i18n="menu.home">${I18n.get('menu.home')}</span>
          </a>
          <a class="sidebar-item ${activeBase === 'services' ? 'active' : ''}" data-page="services">
            ${Icons.get('services')}
            <span data-i18n="menu.services">${I18n.get('menu.services')}</span>
          </a>
        </div>
      </nav>
    `;

    sidebar.querySelectorAll('[data-page]').forEach(el => {
      el.addEventListener('click', (e) => {
        e.preventDefault();
        navigate(el.dataset.page);
      });
    });
  }

  function renderHeader() {
    const header = document.getElementById('header');
    header.innerHTML = `
      <div class="header-left">
        <span class="header-title" id="header-title"></span>
      </div>
      <div class="header-right" id="header-actions">
        <button class="icon-btn" id="settings-btn" title="${I18n.get('menu.settings')}">
          ${Icons.get('settings')}
        </button>
        <button class="icon-btn" id="logout-btn" title="${I18n.get('app.close')}">
          ${Icons.get('logout')}
        </button>
      </div>
    `;

    document.getElementById('settings-btn').addEventListener('click', () => navigate('settings'));
    document.getElementById('logout-btn').addEventListener('click', handleLogout);
  }

  function setHeaderTitle(key) {
    const title = document.getElementById('header-title');
    if (title) title.textContent = I18n.get(key);
  }

  function setHeaderActions(html) {
    const actions = document.getElementById('header-actions');
    if (actions) actions.innerHTML = html;
  }

  function renderCreateMenu(options) {
    const items = options.map(opt => `
      <div class="dropdown-item" data-action="${opt.action}">
        ${opt.icon ? opt.icon : ''}
        <span>${opt.label}</span>
      </div>
    `).join('');

    return `
      <div class="custom-dropdown page-create-menu" id="page-create-menu">
        <button class="icon-btn dropdown-trigger" title="${I18n.get('app.add')}">
          ${Icons.get('more')}
        </button>
        <div class="dropdown-menu">${items}</div>
      </div>
    `;
  }

  function attachCreateMenu(options) {
    const menu = document.getElementById('page-create-menu');
    if (!menu) return;

    const trigger = menu.querySelector('.dropdown-trigger');
    trigger.addEventListener('click', (e) => {
      e.stopPropagation();
      document.querySelectorAll('.custom-dropdown.open').forEach(d => {
        if (d !== menu) d.classList.remove('open');
      });
      menu.classList.toggle('open');
    });

    menu.querySelectorAll('[data-action]').forEach(item => {
      item.addEventListener('click', (e) => {
        e.stopPropagation();
        menu.classList.remove('open');
        const action = item.dataset.action;
        const opt = options.find(o => o.action === action);
        if (opt && opt.onClick) opt.onClick();
      });
    });

    document.addEventListener('click', () => menu.classList.remove('open'), { once: true });
  }

  function renderCardMenu(options) {
    const items = options.map(opt => `
      <div class="dropdown-item ${opt.danger ? 'danger' : ''}" data-action="${opt.action}">
        ${opt.icon ? opt.icon : ''}
        <span>${opt.label}</span>
      </div>
    `).join('');

    return `
      <div class="custom-dropdown card-menu">
        <button class="icon-btn dropdown-trigger">
          ${Icons.get('more')}
        </button>
        <div class="dropdown-menu">${items}</div>
      </div>
    `;
  }

  function attachCardMenus(container, optionsMap) {
    container.querySelectorAll('.card-menu').forEach(menu => {
      const trigger = menu.querySelector('.dropdown-trigger');
      const dropdown = menu.querySelector('.dropdown-menu');
      trigger.addEventListener('click', (e) => {
        e.stopPropagation();
        const isOpen = menu.classList.contains('open');
        document.querySelectorAll('.custom-dropdown.open').forEach(d => {
          if (d !== menu) {
            d.classList.remove('open');
            const dd = d.querySelector('.dropdown-menu');
            if (dd) dd.removeAttribute('style');
          }
        });
        if (!isOpen) {
          menu.classList.add('open');
          const rect = trigger.getBoundingClientRect();
          dropdown.style.position = 'fixed';
          dropdown.style.top = (rect.bottom + 4) + 'px';
          dropdown.style.right = (window.innerWidth - rect.right) + 'px';
          dropdown.style.zIndex = '9999';
        } else {
          menu.classList.remove('open');
          dropdown.removeAttribute('style');
        }
      });

      menu.querySelectorAll('[data-action]').forEach(item => {
        item.addEventListener('click', (e) => {
          e.stopPropagation();
          menu.classList.remove('open');
          dropdown.removeAttribute('style');
          const action = item.dataset.action;
          const card = menu.closest('[data-id]');
          const id = card ? card.dataset.id : null;
          if (optionsMap[action]) optionsMap[action](id);
        });
      });
    });

    document.addEventListener('click', () => {
      document.querySelectorAll('.custom-dropdown.open').forEach(d => {
        d.classList.remove('open');
        const dd = d.querySelector('.dropdown-menu');
        if (dd) dd.removeAttribute('style');
      });
    });
  }

  function renderCustomSelect(name, options, selectedValue, placeholder) {
    const selected = options.find(o => o.value === selectedValue);
    const items = options.map(o => `
      <div class="custom-select-option ${o.value === selectedValue ? 'selected' : ''}" data-value="${o.value}">
        ${o.label}
      </div>
    `).join('');

    return `
      <div class="custom-select" id="select-${name}">
        <input type="hidden" name="${name}" value="${selectedValue || ''}">
        <div class="custom-select-trigger">
          <span class="select-label">${selected ? selected.label : (placeholder || I18n.get('app.filter'))}</span>
          ${Icons.get('chevron')}
        </div>
        <div class="custom-select-menu">${items}</div>
      </div>
    `;
  }

  function attachCustomSelect(name, onChange) {
    const select = document.getElementById(`select-${name}`);
    if (!select) return;

    const trigger = select.querySelector('.custom-select-trigger');
    const input = select.querySelector('input');
    const label = select.querySelector('.select-label');

    trigger.addEventListener('click', (e) => {
      e.stopPropagation();
      document.querySelectorAll('.custom-select.open').forEach(s => {
        if (s !== select) s.classList.remove('open');
      });
      select.classList.toggle('open');
    });

    select.querySelectorAll('.custom-select-option').forEach(opt => {
      opt.addEventListener('click', (e) => {
        e.stopPropagation();
        select.classList.remove('open');
        input.value = opt.dataset.value;
        label.textContent = opt.textContent.trim();
        select.querySelectorAll('.custom-select-option').forEach(o => o.classList.remove('selected'));
        opt.classList.add('selected');
        if (onChange) onChange(opt.dataset.value);
      });
    });

    document.addEventListener('click', () => select.classList.remove('open'));
  }

  // -------------------- Home Page --------------------
  async function renderHomePage() {
    await loadData();
    const content = document.getElementById('content');

    const createOptions = [{
      action: 'create',
      label: I18n.get('home.create'),
      icon: Icons.get('plus'),
      onClick: () => navigate('home/new')
    }];
    setHeaderActions(renderCreateMenu(createOptions));
    attachCreateMenu(createOptions);

    if (state.homes.length === 0) {
      content.innerHTML = renderEmptyCard({
        icon: Icons.get('home'),
        titleKey: 'home.empty',
        subtitleKey: 'home.subtitle',
        actionLabel: I18n.get('home.create'),
        actionIcon: Icons.get('plus'),
        onAction: () => navigate('home/new')
      });
      return;
    }

    content.innerHTML = `
      <div class="page-header">
        <h2 data-i18n="home.title">${I18n.get('home.title')}</h2>
      </div>
      <div class="card-grid" id="homes-grid"></div>
    `;

    const grid = document.getElementById('homes-grid');
    grid.innerHTML = state.homes.map(home => `
      <div class="card" data-id="${home.id}">
        ${renderCardMenu([
          { action: 'edit', label: I18n.get('app.edit'), icon: Icons.get('edit') },
          { action: 'delete', label: I18n.get('app.delete'), icon: Icons.get('delete'), danger: true }
        ])}
        <div class="card-header">
          <div class="card-icon">${Icons.get('home')}</div>
        </div>
        <h3 class="card-title">${escapeHtml(home.name)}</h3>
        <p class="card-subtitle">${home.address ? escapeHtml(home.address) : ''}</p>
      </div>
    `).join('');

    attachCardMenus(grid, {
      edit: (id) => navigate(`home/edit/${id}`),
      delete: (id) => openDeleteModal({
        title: I18n.get('app.confirm'),
        subtitle: `${I18n.get('home.title')}: ${escapeHtml(state.homes.find(h => h.id == id)?.name || '')}`,
        onConfirm: async () => {
          await API.homes.delete(id);
          navigate('home');
        }
      })
    });
  }

  async function renderHomeFormPage(id) {
    await loadData();
    const isEdit = !!id;
    const home = isEdit ? state.homes.find(h => h.id == id) : null;

    setHeaderActions(`
      <button class="icon-btn" id="form-back" title="${I18n.get('app.cancel')}">${Icons.get('cancel')}</button>
    `);

    const content = document.getElementById('content');
    content.innerHTML = `
      <div class="form-page">
        <h2 data-i18n="${isEdit ? 'home.edit' : 'home.create'}">${I18n.get(isEdit ? 'home.edit' : 'home.create')}</h2>
        <form id="home-form">
          <div class="form-group">
            <label data-i18n="home.name">${I18n.get('home.name')}</label>
            <input type="text" name="name" value="${home ? escapeHtml(home.name) : ''}" required>
          </div>
          <div class="form-group">
            <label data-i18n="home.address">${I18n.get('home.address')}</label>
            <input type="text" name="address" value="${home && home.address ? escapeHtml(home.address) : ''}">
          </div>
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" id="form-cancel">${Icons.get('cancel')} ${I18n.get('app.cancel')}</button>
            <button type="submit" class="btn btn-primary">${Icons.get('save')} ${I18n.get('app.save')}</button>
          </div>
        </form>
      </div>
    `;

    document.getElementById('form-back').addEventListener('click', () => navigate('home'));
    document.getElementById('form-cancel').addEventListener('click', () => navigate('home'));
    document.getElementById('home-form').addEventListener('submit', async (e) => {
      e.preventDefault();
      const data = Object.fromEntries(new FormData(e.target));
      if (isEdit) {
        await API.homes.update(home.id, data);
      } else {
        await API.homes.create(data);
      }
      navigate('home');
    });
  }

  // -------------------- Services Page --------------------
  async function renderServicesPage() {
    await loadData();
    await refreshServices();
    const content = document.getElementById('content');

    const svcCreateOptions = [{
      action: 'create',
      label: I18n.get('services.create'),
      icon: Icons.get('plus'),
      onClick: () => navigate('services/new')
    }];
    setHeaderActions(renderCreateMenu(svcCreateOptions));
    attachCreateMenu(svcCreateOptions);

    if (state.homes.length === 0) {
      content.innerHTML = renderEmptyCard({
        icon: Icons.get('home'),
        titleKey: 'services.empty_no_home',
        subtitleKey: 'services.subtitle',
        actionLabel: I18n.get('home.create'),
        actionIcon: Icons.get('plus'),
        onAction: () => navigate('home/new')
      });
      return;
    }

    const homeOptions = [{ value: '', label: I18n.get('services.all_homes') }, ...state.homes.map(h => ({ value: String(h.id), label: h.name }))];

    content.innerHTML = `
      <div class="page-header">
        <h2 data-i18n="services.title">${I18n.get('services.title')}</h2>
      </div>
      <div class="filters">
        ${renderCustomSelect('home_filter', homeOptions, '', I18n.get('services.all_homes'))}
      </div>
      <div class="card-grid" id="services-grid"></div>
    `;

    attachCustomSelect('home_filter', async (value) => {
      await refreshServices(value ? parseInt(value) : null);
      renderServicesGrid();
    });

    renderServicesGrid();
  }

  function renderServicesGrid() {
    const grid = document.getElementById('services-grid');
    if (state.services.length === 0) {
      grid.innerHTML = renderEmptyCard({
        icon: Icons.get('services'),
        titleKey: 'services.empty',
        subtitleKey: 'services.subtitle',
        actionLabel: I18n.get('services.create'),
        actionIcon: Icons.get('plus'),
        onAction: () => navigate('services/new')
      });
      return;
    }

    grid.innerHTML = state.services.map(svc => {
      const currency = state.currencies.find(c => c.id === svc.currency_id);
      const home = state.homes.find(h => h.id === svc.home_id);
      return `
        <div class="card" data-id="${svc.id}">
          ${renderCardMenu([
            { action: 'edit', label: I18n.get('app.edit'), icon: Icons.get('edit') },
            { action: 'delete', label: I18n.get('app.delete'), icon: Icons.get('delete'), danger: true }
          ])}
          <div class="card-header">
            <div class="card-icon">${Icons.get(svc.icon_key || 'other')}</div>
            <span class="badge ${svc.active ? 'badge-paid' : 'badge-pending'}">${svc.active ? I18n.get('bills.status_paid') : I18n.get('bills.status_pending')}</span>
          </div>
          <h3 class="card-title">${escapeHtml(svc.name)}</h3>
          <p class="card-subtitle">${escapeHtml(svc.institution || '')} · ${home ? escapeHtml(home.name) : ''}</p>
          <p class="card-meta">${currency ? currency.symbol : ''}${svc.suggested_amount.toFixed(2)} · ${I18n.get('frequency.' + svc.frequency)}</p>
        </div>
      `;
    }).join('');

    attachCardMenus(grid, {
      edit: (id) => navigate(`services/edit/${id}`),
      delete: (id) => openDeleteModal({
        title: I18n.get('app.confirm'),
        subtitle: `${I18n.get('services.title')}: ${escapeHtml(state.services.find(s => s.id == id)?.name || '')}`,
        onConfirm: async () => {
          await API.services.delete(id);
          navigate('services');
        }
      })
    });

    grid.querySelectorAll('.card').forEach(card => {
      card.addEventListener('click', (e) => {
        if (e.target.closest('.card-menu') || e.target.closest('.dropdown-trigger')) return;
        const id = card.dataset.id;
        navigate(`services/bills/${id}`);
      });
    });
  }

  async function renderServiceFormPage(id) {
    await loadData();
    const isEdit = !!id;
    const svc = isEdit ? state.services.find(s => s.id == id) : null;

    if (!isEdit && state.homes.length === 0) {
      navigate('home/new');
      return;
    }

    setHeaderActions(`
      <button class="icon-btn" id="form-back" title="${I18n.get('app.cancel')}">${Icons.get('cancel')}</button>
    `);

    const homeOptions = state.homes.map(h => ({ value: String(h.id), label: h.name }));
    const currencyOptions = state.currencies.map(c => ({ value: String(c.id), label: `${c.code} (${c.symbol})` }));
    const frequencyOptions = [
      { value: 'monthly', label: I18n.get('frequency.monthly') },
      { value: 'yearly', label: I18n.get('frequency.yearly') }
    ];
    const iconOptions = Icons.names().map(key => `
      <div class="icon-option ${svc && svc.icon_key === key ? 'selected' : ''}" data-icon="${key}">
        ${Icons.get(key)}
      </div>
    `).join('');

    const content = document.getElementById('content');
    content.innerHTML = `
      <div class="form-page">
        <h2 data-i18n="${isEdit ? 'services.edit' : 'services.create'}">${I18n.get(isEdit ? 'services.edit' : 'services.create')}</h2>
        <form id="service-form">
          <div class="form-group">
            <label data-i18n="services.home">${I18n.get('services.home')}</label>
            ${renderCustomSelect('home_id', homeOptions, svc ? String(svc.home_id) : String(state.homes[0].id))}
          </div>
          <div class="form-group">
            <label data-i18n="services.name">${I18n.get('services.name')}</label>
            <input type="text" name="name" value="${svc ? escapeHtml(svc.name) : ''}" required>
          </div>
          <div class="form-group">
            <label data-i18n="services.institution">${I18n.get('services.institution')}</label>
            <input type="text" name="institution" value="${svc && svc.institution ? escapeHtml(svc.institution) : ''}">
          </div>
          <div class="form-row">
            <div class="form-group">
              <label data-i18n="services.currency">${I18n.get('services.currency')}</label>
              ${renderCustomSelect('currency_id', currencyOptions, svc ? String(svc.currency_id) : String(state.currencies[0].id))}
            </div>
            <div class="form-group">
              <label data-i18n="services.frequency">${I18n.get('services.frequency')}</label>
              ${renderCustomSelect('frequency', frequencyOptions, svc ? svc.frequency : 'monthly')}
            </div>
          </div>
          <div class="form-group">
            <label data-i18n="services.suggested_amount">${I18n.get('services.suggested_amount')}</label>
            <input type="number" step="0.01" name="suggested_amount" value="${svc ? svc.suggested_amount : '0.00'}" required>
          </div>
          <div class="form-group">
            <label data-i18n="services.icon">${I18n.get('services.icon')}</label>
            <div class="icon-picker" id="icon-picker">${iconOptions}</div>
            <input type="hidden" name="icon_key" id="icon-key-input" value="${svc ? svc.icon_key : 'other'}" required>
          </div>
          <div class="form-group">
            <label class="toggle">
              <input type="checkbox" name="active" ${svc === null || svc.active ? 'checked' : ''}>
              <span class="toggle-slider"></span>
            </label>
            <span data-i18n="services.active" style="margin-left: 0.5rem; vertical-align: super;">${I18n.get('services.active')}</span>
          </div>
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" id="form-cancel">${Icons.get('cancel')} ${I18n.get('app.cancel')}</button>
            <button type="submit" class="btn btn-primary">${Icons.get('save')} ${I18n.get('app.save')}</button>
          </div>
        </form>
      </div>
    `;

    attachCustomSelect('home_id');
    attachCustomSelect('currency_id');
    attachCustomSelect('frequency');

    document.getElementById('form-back').addEventListener('click', () => navigate('services'));
    document.getElementById('form-cancel').addEventListener('click', () => navigate('services'));

    document.querySelectorAll('#icon-picker .icon-option').forEach(el => {
      el.addEventListener('click', () => {
        document.querySelectorAll('#icon-picker .icon-option').forEach(o => o.classList.remove('selected'));
        el.classList.add('selected');
        document.getElementById('icon-key-input').value = el.dataset.icon;
      });
    });

    document.getElementById('service-form').addEventListener('submit', async (e) => {
      e.preventDefault();
      const form = e.target;
      const data = {
        home_id: parseInt(form.querySelector('[name="home_id"]').value),
        name: form.querySelector('[name="name"]').value,
        institution: form.querySelector('[name="institution"]').value,
        currency_id: parseInt(form.querySelector('[name="currency_id"]').value),
        frequency: form.querySelector('[name="frequency"]').value,
        suggested_amount: parseFloat(form.querySelector('[name="suggested_amount"]').value) || 0,
        active: form.querySelector('[name="active"]').checked,
        icon_key: form.querySelector('[name="icon_key"]').value
      };

      if (isEdit) {
        await API.services.update(svc.id, data);
      } else {
        await API.services.create(data);
      }
      navigate('services');
    });
  }

  // -------------------- Bills Page --------------------
  async function renderBillsPage(serviceId) {
    await loadData();
    const service = state.services.find(s => s.id == serviceId);
    if (!service) {
      navigate('services');
      return;
    }
    const bills = await API.bills.list(serviceId);

    const billCreateOptions = [{
      action: 'create',
      label: I18n.get('bills.create'),
      icon: Icons.get('plus'),
      onClick: () => navigate(`bills/new?service=${serviceId}`)
    }];
    setHeaderActions(renderCreateMenu(billCreateOptions));
    attachCreateMenu(billCreateOptions);

    const content = document.getElementById('content');
    content.innerHTML = `
      <div class="page-header">
        <button class="btn btn-ghost" id="back-services">${Icons.get('back')} ${I18n.get('menu.services')}</button>
        <h2>${escapeHtml(service.name)}</h2>
      </div>
      <div id="bills-container"></div>
    `;

    document.getElementById('back-services').addEventListener('click', () => navigate('services'));

    const container = document.getElementById('bills-container');
    if (bills.length === 0) {
      container.innerHTML = renderEmptyCard({
        icon: Icons.get('bill'),
        titleKey: 'bills.empty',
        subtitleKey: 'bills.subtitle',
        actionLabel: I18n.get('bills.create'),
        actionIcon: Icons.get('plus'),
        onAction: () => navigate(`bills/new?service=${serviceId}`)
      });
      return;
    }

    container.innerHTML = `
      <table class="table">
        <thead>
          <tr>
            <th>${I18n.get('bills.year')}</th>
            <th>${I18n.get('bills.month')}</th>
            <th>${I18n.get('bills.amount')}</th>
            <th>${I18n.get('bills.invoice_number')}</th>
            <th>${I18n.get('bills.status')}</th>
            <th>${I18n.get('bills.drive_url')}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${bills.map(bill => `
            <tr data-id="${bill.id}">
              <td>${bill.year}</td>
              <td>${I18n.get('months.' + bill.month)}</td>
              <td>${bill.amount.toFixed(2)}</td>
              <td>${escapeHtml(bill.invoice_number || '-')}</td>
              <td><span class="badge ${bill.status === 'paid' ? 'badge-paid' : 'badge-pending'}">${I18n.get('bills.status_' + bill.status)}</span></td>
              <td>${bill.drive_url ? `<a href="${escapeHtml(bill.drive_url)}" target="_blank">Drive</a>` : '-'}</td>
              <td>
                ${renderCardMenu([
                  { action: 'edit', label: I18n.get('app.edit'), icon: Icons.get('edit') },
                  { action: 'delete', label: I18n.get('app.delete'), icon: Icons.get('delete'), danger: true }
                ])}
              </td>
            </tr>
          `).join('')}
        </tbody>
      </table>
    `;

    attachCardMenus(container, {
      edit: (id) => navigate(`bills/edit/${id}?service=${serviceId}`),
      delete: (id) => openDeleteModal({
        title: I18n.get('app.confirm'),
        subtitle: `${I18n.get('bills.title')} #${id}`,
        onConfirm: async () => {
          await API.bills.delete(id);
          navigate(`services/bills/${serviceId}`);
        }
      })
    });
  }

  async function renderBillFormPage(id) {
    await loadData();
    const params = new URLSearchParams(window.location.search);
    const serviceId = params.get('service');
    const isEdit = !!id;
    const bill = isEdit ? (await API.bills.get(id)) : null;

    if (!serviceId) {
      navigate('services');
      return;
    }

    const service = state.services.find(s => s.id == serviceId);

    setHeaderActions(`
      <button class="icon-btn" id="form-back" title="${I18n.get('app.cancel')}">${Icons.get('cancel')}</button>
    `);

    const months = Array.from({ length: 12 }, (_, i) => i + 1).map(m => ({
      value: String(m),
      label: I18n.get('months.' + m)
    }));
    const statusOptions = [
      { value: 'pending', label: I18n.get('bills.status_pending') },
      { value: 'paid', label: I18n.get('bills.status_paid') }
    ];

    const content = document.getElementById('content');
    content.innerHTML = `
      <div class="form-page">
        <h2 data-i18n="${isEdit ? 'bills.edit' : 'bills.create'}">${I18n.get(isEdit ? 'bills.edit' : 'bills.create')}</h2>
        <p class="card-subtitle" style="margin-bottom:1.5rem;">${service ? escapeHtml(service.name) : ''}</p>
        <form id="bill-form">
          <input type="hidden" name="service_id" value="${serviceId}">
          <div class="form-row">
            <div class="form-group">
              <label data-i18n="bills.year">${I18n.get('bills.year')}</label>
              <input type="number" name="year" value="${bill ? bill.year : new Date().getFullYear()}" required>
            </div>
            <div class="form-group">
              <label data-i18n="bills.month">${I18n.get('bills.month')}</label>
              ${renderCustomSelect('month', months, String(bill ? bill.month : new Date().getMonth() + 1))}
            </div>
          </div>
          <div class="form-group">
            <label data-i18n="bills.amount">${I18n.get('bills.amount')}</label>
            <input type="number" step="0.01" name="amount" value="${bill ? bill.amount : '0.00'}" required>
          </div>
          <div class="form-group">
            <label data-i18n="bills.invoice_number">${I18n.get('bills.invoice_number')}</label>
            <input type="text" name="invoice_number" value="${bill && bill.invoice_number ? escapeHtml(bill.invoice_number) : ''}">
          </div>
          <div class="form-group">
            <label data-i18n="bills.status">${I18n.get('bills.status')}</label>
            ${renderCustomSelect('status', statusOptions, bill ? bill.status : 'pending')}
          </div>
          <div class="form-group">
            <label data-i18n="bills.drive_url">${I18n.get('bills.drive_url')}</label>
            <input type="url" name="drive_url" id="bill-drive-url" value="${bill && bill.drive_url ? escapeHtml(bill.drive_url) : ''}">
          </div>
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" id="form-cancel">${Icons.get('cancel')} ${I18n.get('app.cancel')}</button>
            <button type="submit" class="btn btn-primary">${Icons.get('save')} ${I18n.get('app.save')}</button>
          </div>
        </form>
      </div>
    `;

    attachCustomSelect('month');
    attachCustomSelect('status', (value) => {
      document.getElementById('bill-drive-url').required = value === 'paid';
    });

    document.getElementById('form-back').addEventListener('click', () => navigate(`services/bills/${serviceId}`));
    document.getElementById('form-cancel').addEventListener('click', () => navigate(`services/bills/${serviceId}`));

    const statusInput = document.querySelector('[name="status"]');
    document.getElementById('bill-drive-url').required = statusInput.value === 'paid';

    document.getElementById('bill-form').addEventListener('submit', async (e) => {
      e.preventDefault();
      const form = e.target;
      const data = {
        service_id: parseInt(form.querySelector('[name="service_id"]').value),
        year: parseInt(form.querySelector('[name="year"]').value),
        month: parseInt(form.querySelector('[name="month"]').value),
        amount: parseFloat(form.querySelector('[name="amount"]').value) || 0,
        invoice_number: form.querySelector('[name="invoice_number"]').value,
        status: form.querySelector('[name="status"]').value,
        drive_url: form.querySelector('[name="drive_url"]').value
      };

      if (isEdit) {
        await API.bills.update(bill.id, data);
      } else {
        await API.bills.create(data);
      }
      navigate(`services/bills/${serviceId}`);
    });
  }

  // -------------------- Settings Page --------------------
  async function renderSettingsPage() {
    await loadData();
    const settings = await API.settings.get();

    setHeaderActions('');

    const content = document.getElementById('content');
    content.innerHTML = `
      <div class="settings-section">
        <div class="settings-section-title" data-i18n="settings.section_general">${I18n.get('settings.section_general')}</div>
        <div class="settings-group">
          <div class="settings-row" id="language-row">
            <div class="settings-row-content">
              <div class="settings-row-title" data-i18n="settings.language.title">${I18n.get('settings.language.title')}</div>
              <div class="settings-row-subtitle" data-i18n="settings.language.subtitle">${I18n.get('settings.language.subtitle')}</div>
            </div>
            <div class="settings-row-value">
              <span id="language-value">${I18n.get('settings.language.' + (settings.language || 'es'))}</span>
              ${Icons.get('chevron')}
            </div>
          </div>
        </div>
      </div>

      <div class="settings-section">
        <div class="settings-section-title" data-i18n="settings.section_currencies">${I18n.get('settings.section_currencies')}</div>
        <div class="settings-group">
          ${state.currencies.map(c => `
            <div class="settings-row" data-currency="${c.id}">
              <div class="settings-row-content">
                <div class="settings-row-title">${escapeHtml(c.code)}</div>
                <div class="settings-row-subtitle">${escapeHtml(c.name)} · ${escapeHtml(c.symbol)}</div>
              </div>
              <div class="settings-row-value">${Icons.get('chevron')}</div>
            </div>
          `).join('')}
          <div class="settings-row" id="add-currency-row">
            <div class="settings-row-content">
              <div class="settings-row-title" data-i18n="settings.currencies.create">${I18n.get('settings.currencies.create')}</div>
            </div>
            <div class="settings-row-value">${Icons.get('plus')}</div>
          </div>
        </div>
      </div>
    `;

    document.getElementById('language-row').addEventListener('click', () => navigate('settings/language'));
    document.getElementById('add-currency-row').addEventListener('click', () => navigate('settings/currency'));
    content.querySelectorAll('[data-currency]').forEach(row => {
      row.addEventListener('click', () => navigate(`settings/currency/${row.dataset.currency}`));
    });
  }

  async function renderLanguagePage() {
    setHeaderActions(`
      <button class="icon-btn" id="form-back" title="${I18n.get('app.cancel')}">${Icons.get('cancel')}</button>
    `);

    const content = document.getElementById('content');
    content.innerHTML = `
      <div class="form-page">
        <h2 data-i18n="settings.language.title">${I18n.get('settings.language.title')}</h2>
        <div class="settings-group">
          <div class="settings-row" data-lang="es">
            <div class="settings-row-content">
              <div class="settings-row-title">${I18n.get('settings.language.es')}</div>
            </div>
            ${state.language === 'es' ? `<div class="settings-row-value">✓</div>` : ''}
          </div>
          <div class="settings-row" data-lang="en">
            <div class="settings-row-content">
              <div class="settings-row-title">${I18n.get('settings.language.en')}</div>
            </div>
            ${state.language === 'en' ? `<div class="settings-row-value">✓</div>` : ''}
          </div>
        </div>
      </div>
    `;

    document.getElementById('form-back').addEventListener('click', () => navigate('settings'));
    content.querySelectorAll('[data-lang]').forEach(row => {
      row.addEventListener('click', async () => {
        const lang = row.dataset.lang;
        await API.settings.setLanguage(lang);
        await I18n.load(lang);
        state.language = lang;
        navigate('settings');
      });
    });
  }

  async function renderCurrencyFormPage(id) {
    await loadData();
    const isEdit = !!id;
    const currency = isEdit ? state.currencies.find(c => c.id == id) : null;

    setHeaderActions(`
      <button class="icon-btn" id="form-back" title="${I18n.get('app.cancel')}">${Icons.get('cancel')}</button>
    `);

    const content = document.getElementById('content');
    content.innerHTML = `
      <div class="form-page">
        <h2 data-i18n="${isEdit ? 'app.edit' : 'settings.currencies.create'}">${I18n.get(isEdit ? 'app.edit' : 'settings.currencies.create')}</h2>
        <form id="currency-form">
          <div class="form-group">
            <label data-i18n="settings.currencies.code">${I18n.get('settings.currencies.code')}</label>
            <input type="text" name="code" value="${currency ? escapeHtml(currency.code) : ''}" required maxlength="3">
          </div>
          <div class="form-group">
            <label data-i18n="settings.currencies.name">${I18n.get('settings.currencies.name')}</label>
            <input type="text" name="name" value="${currency ? escapeHtml(currency.name) : ''}" required>
          </div>
          <div class="form-group">
            <label data-i18n="settings.currencies.symbol">${I18n.get('settings.currencies.symbol')}</label>
            <input type="text" name="symbol" value="${currency ? escapeHtml(currency.symbol) : ''}" required>
          </div>
          <div class="form-actions">
            ${isEdit ? `<button type="button" class="btn btn-danger" id="currency-delete">${Icons.get('delete')} ${I18n.get('app.delete')}</button>` : ''}
            <button type="button" class="btn btn-secondary" id="form-cancel">${Icons.get('cancel')} ${I18n.get('app.cancel')}</button>
            <button type="submit" class="btn btn-primary">${Icons.get('save')} ${I18n.get('app.save')}</button>
          </div>
        </form>
      </div>
    `;

    document.getElementById('form-back').addEventListener('click', () => navigate('settings'));
    document.getElementById('form-cancel').addEventListener('click', () => navigate('settings'));

    if (isEdit) {
      document.getElementById('currency-delete').addEventListener('click', () => openDeleteModal({
        title: I18n.get('app.confirm'),
        subtitle: `${I18n.get('settings.currencies.title')}: ${escapeHtml(currency.code)}`,
        onConfirm: async () => {
          await API.currencies.delete(currency.id);
          navigate('settings');
        }
      }));
    }

    document.getElementById('currency-form').addEventListener('submit', async (e) => {
      e.preventDefault();
      const data = Object.fromEntries(new FormData(e.target));
      if (isEdit) {
        await API.currencies.update(currency.id, data);
      } else {
        await API.currencies.create(data);
      }
      navigate('settings');
    });
  }

  // -------------------- Shared helpers --------------------
  function renderEmptyCard({ icon, titleKey, subtitleKey, actionLabel, actionIcon, onAction }) {
    const html = `
      <div class="empty-card">
        <div class="empty-card-icon">${icon}</div>
        <h3 class="empty-card-title" data-i18n="${titleKey}">${I18n.get(titleKey)}</h3>
        <p class="empty-card-subtitle" data-i18n="${subtitleKey}">${I18n.get(subtitleKey)}</p>
        <button class="empty-card-action" id="empty-action">
          ${actionIcon}
          <span>${actionLabel}</span>
        </button>
      </div>
    `;

    setTimeout(() => {
      const btn = document.getElementById('empty-action');
      if (btn) btn.addEventListener('click', onAction);
    }, 0);

    return html;
  }

  function openDeleteModal({ title, subtitle, onConfirm }) {
    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.id = 'delete-modal';
    overlay.innerHTML = `
      <div class="modal">
        <div class="modal-header">
          <div class="modal-icon">${Icons.get('warning')}</div>
          <h3 class="modal-title">${title}</h3>
          <p class="modal-subtitle">${subtitle}</p>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>Escribe "confirmo" para eliminar</label>
            <input type="text" id="confirm-input" placeholder="confirmo" autocomplete="off">
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" id="modal-cancel">${Icons.get('cancel')} ${I18n.get('app.cancel')}</button>
          <button class="btn btn-danger" id="modal-delete" disabled>${Icons.get('delete')} ${I18n.get('app.delete')}</button>
        </div>
      </div>
    `;

    document.body.appendChild(overlay);

    const confirmInput = document.getElementById('confirm-input');
    const deleteBtn = document.getElementById('modal-delete');

    confirmInput.addEventListener('input', () => {
      deleteBtn.disabled = confirmInput.value.trim().toLowerCase() !== 'confirmo';
    });

    document.getElementById('modal-cancel').addEventListener('click', closeDeleteModal);
    deleteBtn.addEventListener('click', async () => {
      if (confirmInput.value.trim().toLowerCase() !== 'confirmo') return;
      try {
        await onConfirm();
      } catch (err) {
        alert(err.message || I18n.get('errors.generic'));
      }
      closeDeleteModal();
    });

    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) closeDeleteModal();
    });
  }

  function closeDeleteModal() {
    const modal = document.getElementById('delete-modal');
    if (modal) modal.remove();
  }

  function escapeHtml(text) {
    if (text == null) return '';
    return String(text)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  async function handleLogout() {
    try {
      await fetch('/api/logout', { method: 'POST' });
    } catch (err) {
      console.error(err);
    }
    window.location.href = '/login';
  }

  return {
    init,
    navigate
  };
})();

document.addEventListener('DOMContentLoaded', App.init);
