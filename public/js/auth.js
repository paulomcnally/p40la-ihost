function showError(message) {
  const el = document.getElementById('error');
  if (!el) return;
  el.textContent = message;
  el.style.display = 'block';
}

function clearError() {
  const el = document.getElementById('error');
  if (!el) return;
  el.textContent = '';
  el.style.display = 'none';
}

async function handleSetup(event) {
  event.preventDefault();
  clearError();

  const email = document.getElementById('email').value.trim();
  const password = document.getElementById('password').value;
  const passwordConfirm = document.getElementById('password_confirm').value;

  try {
    const res = await fetch('/api/setup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, password_confirm: passwordConfirm }),
    });

    if (res.ok) {
      window.location.href = '/';
      return;
    }

    const data = await res.json().catch(() => ({}));
    showError(data.message || 'Error al configurar el usuario');
  } catch (err) {
    showError('Error de conexión con el servidor');
  }
}

async function handleLogin(event) {
  event.preventDefault();
  clearError();

  const email = document.getElementById('email').value.trim();
  const password = document.getElementById('password').value;
  const remember = document.getElementById('remember')?.checked || false;

  try {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, remember }),
    });

    if (res.ok) {
      window.location.href = '/';
      return;
    }

    const data = await res.json().catch(() => ({}));
    showError(data.message || 'Email o contraseña incorrectos');
  } catch (err) {
    showError('Error de conexión con el servidor');
  }
}

async function handleLogout() {
  try {
    await fetch('/api/logout', { method: 'POST' });
  } catch (err) {
    // Ignorar errores de red; de todos modos redirigimos.
  }
  window.location.href = '/login';
}

function initSetup() {
  const form = document.getElementById('setup-form');
  if (form) form.addEventListener('submit', handleSetup);
}

function initLogin() {
  const form = document.getElementById('login-form');
  if (form) form.addEventListener('submit', handleLogin);
}

function initDashboard() {
  const btn = document.getElementById('logout-btn');
  if (btn) btn.addEventListener('click', handleLogout);
}

if (document.getElementById('setup-form')) initSetup();
if (document.getElementById('login-form')) initLogin();
if (document.getElementById('logout-btn')) initDashboard();
