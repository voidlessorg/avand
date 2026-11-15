const $ = (s, r = document) => r.querySelector(s);
const $$ = (s, r = document) => [...r.querySelectorAll(s)];
const esc = (v) => String(v ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const fa = (n) => Number(n || 0).toLocaleString('fa-IR');
const fmtSize = (b) => {
  b = Number(b || 0); if (b < 1024) return fa(b) + ' بایت';
  const u = ['کیلوبایت', 'مگابایت', 'گیگابایت', 'ترابایت']; let i = -1;
  do { b /= 1024; i++; } while (b >= 1024 && i < 3);
  return b.toLocaleString('fa-IR', { maximumFractionDigits: b < 10 ? 1 : 0 }) + ' ' + u[i];
};
const dateFa = (ts, long) => new Date(ts * 1000).toLocaleDateString('fa-IR', long ? { year: 'numeric', month: 'long', day: 'numeric' } : { year: 'numeric', month: 'short', day: 'numeric' });
const timeFa = (ts) => new Date(ts * 1000).toLocaleString('fa-IR', { dateStyle: 'medium', timeStyle: 'short' });

async function api(path, opt = {}) {
  const o = { method: opt.method || (opt.body !== undefined ? 'POST' : 'GET'), credentials: 'same-origin', headers: {} };
  if (opt.body !== undefined) { o.headers['Content-Type'] = 'application/json'; o.body = JSON.stringify(opt.body); }
  let r, j;
  try { r = await fetch('/api/' + path, o); j = await r.json(); } catch { throw Object.assign(new Error('ارتباط با آوند برقرار نشد.'), { status: 0 }); }
  if (r.status === 401 && !opt.quiet) { location.href = '/'; throw Object.assign(new Error('نشست شما تمام شده است.'), { status: 401 }); }
  if (!j.ok) throw Object.assign(new Error(j.error || 'خطا'), { status: r.status, data: j });
  return j;
}

function toast(msg, kind) {
  let box = $('#toasts'); if (!box) { box = document.createElement('div'); box.id = 'toasts'; document.body.appendChild(box); }
  const t = document.createElement('div'); t.className = 'toast ' + (kind || ''); t.textContent = msg; box.appendChild(t); setTimeout(() => t.remove(), 4200);
}

function modal(html, { wide } = {}) {
  const w = document.createElement('div'); w.className = 'ov';
  w.innerHTML = `<div class="modal ${wide ? 'wide' : ''}" role="dialog" aria-modal="true">${html}</div>`;
  const onKey = (e) => { if (e.key === 'Escape') w.close(); };
  w.close = () => { w.remove(); document.removeEventListener('keydown', onKey); };
  document.addEventListener('keydown', onKey);
  w.addEventListener('mousedown', (e) => { if (e.target === w) w.close(); });
  document.body.appendChild(w);
  $$('[data-close]', w).forEach((b) => b.addEventListener('click', () => w.close()));
  return w;
}
const confirmBox = (msg, okText = 'تأیید', danger) => new Promise((res) => {
  const m = modal(`<p style="margin:0 0 18px;line-height:1.9">${esc(msg)}</p><div class="row"><button class="btn grow" data-close>انصراف</button><button class="btn ${danger ? 'danger' : 'pri'} grow" id="y">${esc(okText)}</button></div>`);
  $('#y', m).onclick = () => { m.close(); res(true); };
  const c = m.close; m.close = () => { c(); res(false); };
});

// تم روشن/تیره (به‌ازای هر دستگاه)
const theme = {
  get() { try { return localStorage.getItem('avand_theme') || 'dark'; } catch { return 'dark'; } },
  apply(t) { document.documentElement.dataset.theme = t; },
  set(t) { try { localStorage.setItem('avand_theme', t); } catch {} this.apply(t); },
  toggle() { this.set(this.get() === 'dark' ? 'light' : 'dark'); },
};
theme.apply(theme.get());

const SUPPORT_URL = 'https://daramet.com/zohalgame';
const supportCard = () => `<a class="support" href="${SUPPORT_URL}" target="_blank" rel="noopener noreferrer"><b>${ic('heart')} از آوند حمایت کنید</b><small>آوند رایگان است و با حمایت شما ادامه پیدا می‌کند.<br><span class="ltr">daramet.com/zohalgame</span></small></a>`;

// صفحه‌ی آغازین: همیشه یک‌بار در هر نشست مرورگر نمایش داده می‌شود (کوتاه، با کلیک بسته می‌شود)
(() => {
  let fin; window.splashDone = new Promise((r) => { fin = r; });
  try { if (sessionStorage.getItem('avand_splash') === '1') { fin(); return; } sessionStorage.setItem('avand_splash', '1'); } catch {}
  const show = () => {
    const d = document.createElement('div'); d.className = 'splash'; d.setAttribute('role', 'presentation');
    d.innerHTML = `<div class="sp-logo">${avandLogo(92)}</div><div class="sp-name">آوند</div>
<div class="sp-by">این نرم‌افزار از مجموعه</div>
<svg class="sp-hand" width="330" height="82" viewBox="0 0 330 82" aria-label="انجام وظیفه"><text x="165" y="58" text-anchor="middle">«انجام وظیفه»</text></svg>
<div class="sp-by" style="margin-top:-14px">است؛ به جهت از بین بردن رانت آپلودسنترهای ایرانی&nbsp;;)</div>
<div class="sp-bar"><i></i></div><div class="sp-hint">برای ادامه کلیک کنید</div>`;
    let gone = false; const bye = () => { if (gone) return; gone = true; d.classList.add('out'); setTimeout(() => { d.remove(); fin(); }, 600); };
    d.addEventListener('click', bye); document.addEventListener('keydown', bye, { once: true }); setTimeout(bye, 3600);
    document.body.appendChild(d);
  };
  if (document.body) show(); else document.addEventListener('DOMContentLoaded', show);
})();
