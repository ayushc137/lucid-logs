// SPA mode for the all-in-one static build (VITE_SPA=1): prerender every route
// as a pure client-side app with no server-side rendering, so the whole UI can
// be served as static files by the Go backend (or any static host).
export const ssr = false;
export const prerender = true;
