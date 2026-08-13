const App = (function () {
  const state = {
    currentPage: 'home',
    homes: [],
    currencies: [],
    services: [],
    homeFilter: null,
    language: 'es'
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
    await loadData();
    navigate('home');

    document.getElementById('logout-btn')?.addEventListener('click', handleLogout);
  }

  async function loadData() {
    try {
      const [homes, currencies] = await Promise.all([
        API.homes.list(),
        API.currencies.list()
      ]);
      state.homes = homes || [];
      state.currencies = currencies || [];
    } catch (err) {
      console.error('loadData error:', err);
    }
  }

  async function refreshServices() {
    try {
      state.services = await API.services.list(state.homeFilter) || [];
    } catch (err) {
      console.error('refreshServices error:', err);
      state.services = [];
    }
  }

  function renderSidebar() {
    const sidebar = document.getElementById('sidebar');
    sidebar.innerHTML = `
      <div class="sidebar-brand">${I18n.get('app.title')}</div>
      <nav class="sidebar-nav">
        <div class="sidebar-menu">
          <div class="sidebar-menu-title">${I18n.get('app.title')}</div>
          <a class="sidebar-item ${state.currentPage === 'home' ? 'active' : ''}" data-page="home">
            ${Icons.get('home')}
            <span data-i18n="menu.home">${I18n.get('menu.home')}</span>
          </a>
          <a class="sidebar-item ${state.currentPage === 'services' ? 'active' : ''}" data-page="services">
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
      <div class="header-title" id="header-title"></div>
      <div class="header-actions">
        <button class="icon-btn" id="settings-btn" title="${I18n.get('menu.settings')}">
          ${Icons.get('settings')}
        </button>
        <button class="icon-btn" id="logout-btn" title="${I18n.get('app.close')}">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path><polyline points="16 17 21 12 16 7"></polyline><line x1="21" y1="12" x2="9" y2="12"></line></svg>
        </button>
      </div>
    `;

    document.getElementById('settings-btn').addEventListener('click', () => navigate('settings'));
  }

  function setHeaderTitle(key) {
    const title = document.getElementById('header-title');
    if (title) title.textContent = I18n.get(key);
  }

  function navigate(page) {
    state.currentPage = page;
    renderSidebar();
    const content = document.getElementById('content');
    content.innerHTML = `<div class="empty-state">${I18n.get('app.loading')}</div>`;

    switch (page) {
      case 'home':
        renderHomePage();
        break;
      case 'services':
        renderServicesPage();
        break;
      case 'settings':
        renderSettingsPage();
        break;
      default:
        renderHomePage();
    }
  }

  async function handleLogout() {
    try {
      await fetch('/api/logout', { method: 'POST' });
    } catch (err) {
      console.error(err);
    }
    window.location.href = '/login';
  }

  // -------------------- Home Page --------------------
  async function renderHomePage() {
    setHeaderTitle('home.title');
    await loadData();

    const content = document.getElementById('content');
    if (state.homes.length === 0) {
      content.innerHTML = renderEmptyState('home.empty', () => openHomeModal());
      attachEmptyCreate(content, () => openHomeModal());
      return;
    }

    content.innerHTML = `
      <div class="filters">
        <button class="btn btn-primary" id="add-home-btn">
          ${Icons.get('plus')}
          <span data-i18n="home.create">${I18n.get('home.create')}</span>
        </button>
      </div>
      <div class="card-grid" id="homes-grid"></div>
    `;

    document.getElementById('add-home-btn').addEventListener('click', () => openHomeModal());

    const grid = document.getElementById('homes-grid');
    grid.innerHTML = state.homes.map(home => `
      <div class="card">
        <div class="card-header">
          <div class="card-icon">${Icons.get('home')}</div>
        </div>
        <h3 class="card-title">${escapeHtml(home.name)}</h3>
        <p class="card-subtitle">${home.address ? escapeHtml(home.address) : ''}</p>
        <div class="card-actions">
          <button class="btn btn-secondary" data-edit="${home.id}">
            ${Icons.get('edit')}
            <span data-i18n="app.edit">${I18n.get('app.edit')}</span>
          </button>
          <button class="btn btn-danger" data-delete="${home.id}">
            <span data-i18n="app.delete">${I18n.get('app.delete')}</span>
          </button>
        </div>
      </div>
    `).join('');

    grid.querySelectorAll('[data-edit]').forEach(btn => {
      btn.addEventListener('click', () => {
        const home = state.homes.find(h => h.id == btn.dataset.edit);
        openHomeModal(home);
      });
    });

    grid.querySelectorAll('[data-delete]').forEach(btn => {
      btn.addEventListener('click', () => deleteHome(btn.dataset.delete));
    });
  }

  function openHomeModal(home = null) {
    const isEdit = !!home;
    openModal({
      title: I18n.get(isEdit ? 'home.edit' : 'home.create'),
      body: `
        <form id="home-form">
          <div class="form-group">
            <label data-i18n="home.name">${I18n.get('home.name')}</label>
            <input type="text" name="name" value="${home ? escapeHtml(home.name) : ''}" required>
          </div>
          <div class="form-group">
            <label data-i18n="home.address">${I18n.get('home.address')}</label>
            <input type="text" name="address" value="${home && home.address ? escapeHtml(home.address) : ''}">
          </div>
        </form>
      `,
      onSave: async () => {
        const form = document.getElementById('home-form');
        const data = Object.fromEntries(new FormData(form));
        if (isEdit) {
          await API.homes.update(home.id, data);
        } else {
          await API.homes.create(data);
        }
        closeModal();
        await renderHomePage();
      }
    });
  }

  async function deleteHome(id) {
    if (!confirm(I18n.get('app.confirm'))) return;
    await API.homes.delete(id);
    await renderHomePage();
  }

  // -------------------- Services Page --------------------
  async function renderServicesPage() {
    setHeaderTitle('services.title');
    await Promise.all([loadData(), refreshServices()]);

    const content = document.getElementById('content');
    const homeCount = state.homes.length;

    if (homeCount === 0) {
      content.innerHTML = `
        <div class="empty-state">
          <p data-i18n="services.empty_no_home">${I18n.get('services.empty_no_home')}</p>
          <button class="btn btn-primary" onclick="App.navigate('home')">${I18n.get('home.create')}</button>
        </div>
      `;
      return;
    }

    content.innerHTML = `
      <div class="filters">
        <button class="btn btn-primary" id="add-service-btn">
          ${Icons.get('plus')}
          <span data-i18n="services.create">${I18n.get('services.create')}</span>
        </button>
        <select id="home-filter">
          <option value="">${I18n.get('services.all_homes')}</option>
          ${state.homes.map(h => `<option value="${h.id}" ${state.homeFilter == h.id ? 'selected' : ''}>${escapeHtml(h.name)}</option>`).join('')}
        </select>
      </div>
      <div class="card-grid" id="services-grid"></div>
    `;

    document.getElementById('add-service-btn').addEventListener('click', () => openServiceModal());
    document.getElementById('home-filter').addEventListener('change', async (e) => {
      state.homeFilter = e.target.value ? parseInt(e.target.value) : null;
      await refreshServices();
      renderServicesGrid();
    });

    renderServicesGrid();
  }

  function renderServicesGrid() {
    const grid = document.getElementById('services-grid');
    if (state.services.length === 0) {
      grid.innerHTML = renderEmptyState('services.empty', () => openServiceModal());
      attachEmptyCreate(grid, () => openServiceModal());
      return;
    }

    grid.innerHTML = state.services.map(svc => {
      const currency = state.currencies.find(c => c.id === svc.currency_id);
      const home = state.homes.find(h => h.id === svc.home_id);
      return `
        <div class="card">
          <div class="card-header">
            <div class="card-icon">${Icons.get(svc.icon_key || 'other')}</div>
            <span class="badge ${svc.active ? 'badge-paid' : 'badge-pending'}">${svc.active ? 'Active' : 'Inactive'}</span>
          </div>
          <h3 class="card-title">${escapeHtml(svc.name)}</h3>
          <p class="card-subtitle">
            ${escapeHtml(svc.institution || '')} · ${home ? escapeHtml(home.name) : ''}
          </p>
          <p class="card-subtitle">
            ${currency ? currency.symbol : ''}${svc.suggested_amount.toFixed(2)} · ${I18n.get('frequency.' + svc.frequency)}
          </p>
          <div class="card-actions">
            <button class="btn btn-secondary" data-edit="${svc.id}">
              ${Icons.get('edit')}
              <span data-i18n="app.edit">${I18n.get('app.edit')}</span>
            </button>
            <button class="btn btn-primary" data-bills="${svc.id}">
              ${Icons.get('bill')}
              <span data-i18n="services.bills">${I18n.get('services.bills')}</span>
            </button>
            <button class="btn btn-danger" data-delete="${svc.id}">
              <span data-i18n="app.delete">${I18n.get('app.delete')}</span>
            </button>
          </div>
        </div>
      `;
    }).join('');

    grid.querySelectorAll('[data-edit]').forEach(btn => {
      btn.addEventListener('click', () => {
        const svc = state.services.find(s => s.id == btn.dataset.edit);
        openServiceModal(svc);
      });
    });

    grid.querySelectorAll('[data-bills]').forEach(btn => {
      btn.addEventListener('click', () => renderBillsPage(btn.dataset.bills));
    });

    grid.querySelectorAll('[data-delete]').forEach(btn => {
      btn.addEventListener('click', () => deleteService(btn.dataset.delete));
    });
  }

  function openServiceModal(svc = null) {
    const isEdit = !!svc;
    const homeOptions = state.homes.map(h =>
      `<option value="${h.id}" ${svc && svc.home_id === h.id ? 'selected' : ''}>${escapeHtml(h.name)}</option>`
    ).join('');
    const currencyOptions = state.currencies.map(c =>
      `<option value="${c.id}" ${svc && svc.currency_id === c.id ? 'selected' : ''}>${escapeHtml(c.code)} (${escapeHtml(c.symbol)})</option>`
    ).join('');
    const iconOptions = Icons.names().map(key => `
      <div class="icon-option ${svc && svc.icon_key === key ? 'selected' : ''}" data-icon="${key}">
        ${Icons.get(key)}
      </div>
    `).join('');

    openModal({
      title: I18n.get(isEdit ? 'services.edit' : 'services.create'),
      body: `
        <form id="service-form">
          <div class="form-group">
            <label data-i18n="services.home">${I18n.get('services.home')}</label>
            <select name="home_id" required>${homeOptions}</select>
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
              <select name="currency_id" required>${currencyOptions}</select>
            </div>
            <div class="form-group">
              <label data-i18n="services.frequency">${I18n.get('services.frequency')}</label>
              <select name="frequency" required>
                <option value="monthly" ${svc && svc.frequency === 'monthly' ? 'selected' : ''}>${I18n.get('frequency.monthly')}</option>
                <option value="yearly" ${svc && svc.frequency === 'yearly' ? 'selected' : ''}>${I18n.get('frequency.yearly')}</option>
              </select>
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
        </form>
      `,
      onSave: async () => {
        const form = document.getElementById('service-form');
        const data = Object.fromEntries(new FormData(form));
        data.home_id = parseInt(data.home_id);
        data.currency_id = parseInt(data.currency_id);
        data.suggested_amount = parseFloat(data.suggested_amount) || 0;
        data.active = !!form.querySelector('[name="active"]').checked;

        if (isEdit) {
          await API.services.update(svc.id, data);
        } else {
          await API.services.create(data);
        }
        closeModal();
        await renderServicesPage();
      }
    });

    document.querySelectorAll('#icon-picker .icon-option').forEach(el => {
      el.addEventListener('click', () => {
        document.querySelectorAll('#icon-picker .icon-option').forEach(o => o.classList.remove('selected'));
        el.classList.add('selected');
        document.getElementById('icon-key-input').value = el.dataset.icon;
      });
    });
  }

  async function deleteService(id) {
    if (!confirm(I18n.get('app.confirm'))) return;
    await API.services.delete(id);
    await renderServicesPage();
  }

  // -------------------- Bills Page --------------------
  async function renderBillsPage(serviceId) {
    setHeaderTitle('bills.title');
    const service = state.services.find(s => s.id == serviceId) || await API.services.get(serviceId);
    const bills = await API.bills.list(serviceId);

    const content = document.getElementById('content');
    content.innerHTML = `
      <div class="filters">
        <button class="btn btn-secondary" id="back-services-btn">
          <span data-i18n="menu.services">${I18n.get('menu.services')}</span>
        </button>
        <button class="btn btn-primary" id="add-bill-btn">
          ${Icons.get('plus')}
          <span data-i18n="bills.create">${I18n.get('bills.create')}</span>
        </button>
      </div>
      <h2 style="margin-top:0;">${escapeHtml(service.name)}</h2>
      <div id="bills-container"></div>
    `;

    document.getElementById('back-services-btn').addEventListener('click', () => renderServicesPage());
    document.getElementById('add-bill-btn').addEventListener('click', () => openBillModal(serviceId));

    renderBillsTable(bills, serviceId);
  }

  function renderBillsTable(bills, serviceId) {
    const container = document.getElementById('bills-container');
    if (bills.length === 0) {
      container.innerHTML = renderEmptyState('bills.empty', () => openBillModal(serviceId));
      attachEmptyCreate(container, () => openBillModal(serviceId));
      return;
    }

    container.innerHTML = `
      <table class="table">
        <thead>
          <tr>
            <th data-i18n="bills.year">${I18n.get('bills.year')}</th>
            <th data-i18n="bills.month">${I18n.get('bills.month')}</th>
            <th data-i18n="bills.amount">${I18n.get('bills.amount')}</th>
            <th data-i18n="bills.invoice_number">${I18n.get('bills.invoice_number')}</th>
            <th data-i18n="bills.status">${I18n.get('bills.status')}</th>
            <th data-i18n="bills.drive_url">${I18n.get('bills.drive_url')}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${bills.map(bill => `
            <tr>
              <td>${bill.year}</td>
              <td>${I18n.get('months.' + bill.month)}</td>
              <td>${bill.amount.toFixed(2)}</td>
              <td>${escapeHtml(bill.invoice_number || '-')}</td>
              <td><span class="badge ${bill.status === 'paid' ? 'badge-paid' : 'badge-pending'}">${I18n.get('bills.status_' + bill.status)}</span></td>
              <td>${bill.drive_url ? `<a href="${escapeHtml(bill.drive_url)}" target="_blank">Drive</a>` : '-'}</td>
              <td>
                <button class="btn btn-secondary" data-edit-bill="${bill.id}">${I18n.get('app.edit')}</button>
                <button class="btn btn-danger" data-delete-bill="${bill.id}">${I18n.get('app.delete')}</button>
              </td>
            </tr>
          `).join('')}
        </tbody>
      </table>
    `;

    container.querySelectorAll('[data-edit-bill]').forEach(btn => {
      btn.addEventListener('click', () => openBillModal(serviceId, bills.find(b => b.id == btn.dataset.editBill)));
    });

    container.querySelectorAll('[data-delete-bill]').forEach(btn => {
      btn.addEventListener('click', () => deleteBill(btn.dataset.deleteBill, serviceId));
    });
  }

  function openBillModal(serviceId, bill = null) {
    const isEdit = !!bill;
    const months = Array.from({ length: 12 }, (_, i) => i + 1).map(m =>
      `<option value="${m}" ${bill && bill.month === m ? 'selected' : ''}>${I18n.get('months.' + m)}</option>`
    ).join('');

    openModal({
      title: I18n.get(isEdit ? 'bills.edit' : 'bills.create'),
      body: `
        <form id="bill-form">
          <input type="hidden" name="service_id" value="${serviceId}">
          <div class="form-row">
            <div class="form-group">
              <label data-i18n="bills.year">${I18n.get('bills.year')}</label>
              <input type="number" name="year" value="${bill ? bill.year : new Date().getFullYear()}" required>
            </div>
            <div class="form-group">
              <label data-i18n="bills.month">${I18n.get('bills.month')}</label>
              <select name="month" required>${months}</select>
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
            <select name="status" id="bill-status" required>
              <option value="pending" ${bill && bill.status === 'pending' ? 'selected' : ''}>${I18n.get('bills.status_pending')}</option>
              <option value="paid" ${bill && bill.status === 'paid' ? 'selected' : ''}>${I18n.get('bills.status_paid')}</option>
            </select>
          </div>
          <div class="form-group">
            <label data-i18n="bills.drive_url">${I18n.get('bills.drive_url')}</label>
            <input type="url" name="drive_url" id="bill-drive-url" value="${bill && bill.drive_url ? escapeHtml(bill.drive_url) : ''}">
          </div>
        </form>
      `,
      onSave: async () => {
        const form = document.getElementById('bill-form');
        const data = Object.fromEntries(new FormData(form));
        data.service_id = parseInt(data.service_id);
        data.year = parseInt(data.year);
        data.month = parseInt(data.month);
        data.amount = parseFloat(data.amount) || 0;

        if (isEdit) {
          await API.bills.update(bill.id, data);
        } else {
          await API.bills.create(data);
        }
        closeModal();
        await renderBillsPage(serviceId);
      }
    });

    const statusSelect = document.getElementById('bill-status');
    const driveInput = document.getElementById('bill-drive-url');
    statusSelect.addEventListener('change', () => {
      driveInput.required = statusSelect.value === 'paid';
    });
    driveInput.required = statusSelect.value === 'paid';
  }

  async function deleteBill(id, serviceId) {
    if (!confirm(I18n.get('app.confirm'))) return;
    await API.bills.delete(id);
    await renderBillsPage(serviceId);
  }

  // -------------------- Settings Page --------------------
  async function renderSettingsPage() {
    setHeaderTitle('settings.title');
    const settings = await API.settings.get();

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

    document.getElementById('language-row').addEventListener('click', openLanguageModal);
    document.getElementById('add-currency-row').addEventListener('click', () => openCurrencyModal());
    content.querySelectorAll('[data-currency]').forEach(row => {
      row.addEventListener('click', () => {
        const currency = state.currencies.find(c => c.id == row.dataset.currency);
        openCurrencyModal(currency);
      });
    });
  }

  function openLanguageModal() {
    openModal({
      title: I18n.get('settings.language.title'),
      body: `
        <div class="settings-group">
          <div class="settings-row" data-lang="es">
            <div class="settings-row-content">
              <div class="settings-row-title">${I18n.get('settings.language.es')}</div>
            </div>
          </div>
          <div class="settings-row" data-lang="en">
            <div class="settings-row-content">
              <div class="settings-row-title">${I18n.get('settings.language.en')}</div>
            </div>
          </div>
        </div>
      `,
      hideFooter: true
    });

    document.querySelectorAll('[data-lang]').forEach(row => {
      row.addEventListener('click', async () => {
        const lang = row.dataset.lang;
        await API.settings.setLanguage(lang);
        await I18n.load(lang);
        state.language = lang;
        closeModal();
        renderSidebar();
        renderHeader();
        await renderSettingsPage();
      });
    });
  }

  function openCurrencyModal(currency = null) {
    const isEdit = !!currency;
    openModal({
      title: I18n.get(isEdit ? 'app.edit' : 'settings.currencies.create'),
      body: `
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
        </form>
      `,
      onSave: async () => {
        const form = document.getElementById('currency-form');
        const data = Object.fromEntries(new FormData(form));
        if (isEdit) {
          await API.currencies.update(currency.id, data);
        } else {
          await API.currencies.create(data);
        }
        await loadData();
        closeModal();
        await renderSettingsPage();
      },
      onDelete: isEdit ? async () => {
        if (!confirm(I18n.get('app.confirm'))) return;
        await API.currencies.delete(currency.id);
        await loadData();
        closeModal();
        await renderSettingsPage();
      } : null
    });
  }

  // -------------------- Modal helpers --------------------
  function openModal({ title, body, onSave, onDelete, hideFooter }) {
    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.id = 'modal-overlay';
    overlay.innerHTML = `
      <div class="modal">
        <div class="modal-header">
          <h3 class="modal-title">${title}</h3>
          <button class="icon-btn" id="modal-close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg></button>
        </div>
        <div class="modal-body">${body}</div>
        ${hideFooter ? '' : `
          <div class="modal-footer">
            ${onDelete ? `<button class="btn btn-danger" id="modal-delete">${I18n.get('app.delete')}</button>` : ''}
            <button class="btn btn-secondary" id="modal-cancel">${I18n.get('app.cancel')}</button>
            <button class="btn btn-primary" id="modal-save">${I18n.get('app.save')}</button>
          </div>
        `}
      </div>
    `;

    document.body.appendChild(overlay);

    document.getElementById('modal-close')?.addEventListener('click', closeModal);
    document.getElementById('modal-cancel')?.addEventListener('click', closeModal);
    document.getElementById('modal-save')?.addEventListener('click', async () => {
      try {
        await onSave();
      } catch (err) {
        alert(err.message || I18n.get('errors.generic'));
      }
    });
    document.getElementById('modal-delete')?.addEventListener('click', async () => {
      try {
        await onDelete();
      } catch (err) {
        alert(err.message || I18n.get('errors.generic'));
      }
    });

    overlay.addEventListener('click', (e) => {
      if (e.target === overlay) closeModal();
    });
  }

  function closeModal() {
    const overlay = document.getElementById('modal-overlay');
    if (overlay) overlay.remove();
  }

  // -------------------- Utilities --------------------
  function renderEmptyState(messageKey, onCreate) {
    return `
      <div class="empty-state">
        ${Icons.get('other')}
        <p data-i18n="${messageKey}">${I18n.get(messageKey)}</p>
        ${onCreate ? `<button class="btn btn-primary empty-create-btn">${I18n.get('app.create')}</button>` : ''}
      </div>
    `;
  }

  function attachEmptyCreate(container, onCreate) {
    const btn = container.querySelector('.empty-create-btn');
    if (btn) btn.addEventListener('click', onCreate);
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

  return {
    init,
    navigate
  };
})();

document.addEventListener('DOMContentLoaded', App.init);
