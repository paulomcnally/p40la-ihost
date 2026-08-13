const API = (function () {
  async function request(path, options = {}) {
    const res = await fetch(path, {
      headers: { 'Content-Type': 'application/json', ...options.headers },
      ...options
    });
    if (res.status === 401) {
      window.location.href = '/login';
      return;
    }
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      const err = new Error(data.message || 'Request failed');
      err.code = data.error || 'error';
      err.status = res.status;
      throw err;
    }
    if (res.status === 204) return null;
    return res.json();
  }

  const get = (path) => request(path, { method: 'GET' });
  const post = (path, body) => request(path, { method: 'POST', body: JSON.stringify(body) });
  const put = (path, body) => request(path, { method: 'PUT', body: JSON.stringify(body) });
  const del = (path) => request(path, { method: 'DELETE' });

  const settings = {
    get: () => get('/api/settings'),
    setLanguage: (language) => post('/api/settings/language', { language })
  };

  const currencies = {
    list: () => get('/api/currencies'),
    create: (body) => post('/api/currencies', body),
    update: (id, body) => put(`/api/currencies/${id}`, body),
    delete: (id) => del(`/api/currencies/${id}`)
  };

  const homes = {
    list: () => get('/api/homes'),
    count: () => get('/api/homes/count'),
    get: (id) => get(`/api/homes/${id}`),
    create: (body) => post('/api/homes', body),
    update: (id, body) => put(`/api/homes/${id}`, body),
    delete: (id) => del(`/api/homes/${id}`)
  };

  const services = {
    list: (homeId) => get(homeId ? `/api/services?home_id=${homeId}` : '/api/services'),
    get: (id) => get(`/api/services/${id}`),
    create: (body) => post('/api/services', body),
    update: (id, body) => put(`/api/services/${id}`, body),
    delete: (id) => del(`/api/services/${id}`)
  };

  const bills = {
    list: (serviceId) => get(`/api/services/${serviceId}/bills`),
    get: (id) => get(`/api/bills/${id}`),
    create: (body) => post('/api/bills', body),
    update: (id, body) => put(`/api/bills/${id}`, body),
    delete: (id) => del(`/api/bills/${id}`)
  };

  return { settings, currencies, homes, services, bills };
})();
