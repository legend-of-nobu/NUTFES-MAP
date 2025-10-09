let csrfToken: string | null = null;

export const fetchWithCsrf = async (url: string, options: RequestInit = {}) => {
  if (!csrfToken) {
    const res = await fetch(`${process.env.NEXT_PUBLIC_API_BASE}/auth/csrf`, {
      credentials: "include",
    });
    if (res.ok && res.status === 200) {
      const data = await res.json();
      csrfToken = data.csrfToken;
    }
  }

  const headers = {
    ...(options.headers || {}),
    ...(csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
  };

  return fetch(url, { ...options, headers });
};
