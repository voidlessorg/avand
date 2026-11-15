/* VoidCaptcha — بررسی مرورگر + پازل انسان‌سنج آوند */
(function () {
  const css = `
.vc-ov{position:fixed;inset:0;z-index:2147483000;display:flex;align-items:center;justify-content:center;padding:16px;background:radial-gradient(1200px 700px at 50% 0%,rgba(20,184,166,.22),transparent 60%),rgba(5,3,13,.92);backdrop-filter:blur(10px);font-family:'Vazirmatn','Segoe UI',Tahoma,'AvandFallback',sans-serif;color:#e4e1ed;direction:rtl;animation:vcin .25s ease-out}
.vc-ov.vc-full{background:radial-gradient(900px 600px at 50% 35%,rgba(20,184,166,.28),transparent 65%),#04100f;backdrop-filter:none}
@keyframes vcin{from{opacity:0}to{opacity:1}}
.vc-out{animation:vcout .45s ease-in forwards}@keyframes vcout{to{opacity:0;visibility:hidden}}
.vc-ov,.vc-card,.vc-handle{-webkit-user-select:none;user-select:none;-webkit-touch-callout:none}
.vc-card{width:100%;max-width:372px;background:linear-gradient(180deg,#0f1f1f,#081413);border:1px solid rgba(20,184,166,.38);border-radius:20px;padding:18px;box-shadow:0 0 0 1px rgba(255,255,255,.03),0 24px 80px rgba(0,0,0,.65),0 0 60px rgba(20,184,166,.22)}
.vc-top{display:flex;align-items:center;gap:12px;margin-bottom:14px}
.vc-top b{display:block;font-size:17px;color:#5eead4}.vc-pre{color:#f5b83d!important;font-weight:600;font-size:11.5px!important;white-space:nowrap}.vc-top small{display:block;font-size:12px;color:#958da1;margin-top:2px}
.vc-x{margin:0;background:none;border:0;color:#958da1;font-size:20px;cursor:pointer;line-height:1;padding:4px 8px;border-radius:8px}.vc-x:hover{background:rgba(255,255,255,.07);color:#fff}
.vc-stage{position:relative;width:320px;height:160px;max-width:100%;margin:0 auto;border-radius:12px;overflow:hidden;direction:ltr;background:#0a1514;box-shadow:inset 0 0 0 1px rgba(255,255,255,.08);user-select:none}
.vc-stage img{position:absolute;display:block;-webkit-user-drag:none;pointer-events:none}
.vc-bg{left:0;top:0;width:320px;height:160px}
.vc-piece{width:60px;height:60px;filter:drop-shadow(0 3px 6px rgba(0,0,0,.55));transition:none}
.vc-msg{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;font-weight:700;font-size:15px;opacity:0;transition:opacity .2s;background:rgba(5,3,13,.55);direction:rtl;pointer-events:none}
.vc-msg.on{opacity:1}.vc-ok{color:#f5b83d}.vc-bad{color:#ffb4ab}
.vc-track{position:relative;width:320px;max-width:100%;height:44px;margin:14px auto 0;border-radius:12px;background:#1b1b23;direction:ltr;box-shadow:inset 0 0 0 1px rgba(255,255,255,.07);overflow:hidden}
.vc-fill{position:absolute;left:0;top:0;bottom:0;width:0;background:linear-gradient(90deg,rgba(20,184,166,.15),rgba(20,184,166,.5))}
.vc-hint{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;font-size:12.5px;color:#958da1;pointer-events:none;direction:rtl}
.vc-handle{position:absolute;left:0;top:0;width:44px;height:44px;border-radius:12px;display:flex;align-items:center;justify-content:center;cursor:grab;touch-action:none;background:linear-gradient(135deg,#14b8a6,#38bdf8);color:#fff;box-shadow:0 0 18px rgba(20,184,166,.6)}
.vc-handle:active{cursor:grabbing}
.vc-shake{animation:vcsh .4s}@keyframes vcsh{20%,60%{transform:translateX(-7px)}40%,80%{transform:translateX(7px)}}
.vc-foot{display:flex;justify-content:center;margin-top:14px}
.vc-ref{margin-right:auto;background:none;border:0;color:#adc6ff;font:inherit;font-size:20px;line-height:1;cursor:pointer;padding:4px 8px;border-radius:8px}.vc-ref:hover{background:rgba(173,198,255,.1)}
.vc-word{font-family:'Space Grotesk','Vazirmatn',sans-serif;font-weight:500;letter-spacing:.02em;direction:ltr;unicode-bidi:isolate}
.vc-word b{font-weight:800;background:linear-gradient(90deg,#5eead4,#38bdf8,#f5b83d);-webkit-background-clip:text;background-clip:text;color:transparent}
.vc-badge{display:inline-flex;align-items:center;gap:8px;font-size:12px;color:#958da1;direction:rtl;padding:5px 12px 5px 10px;border-radius:999px;background:rgba(20,184,166,.1);border:1px solid rgba(20,184,166,.28);font-family:'Vazirmatn','Segoe UI',Tahoma,'AvandFallback',sans-serif;white-space:nowrap}
.vc-badge .vc-word{color:#cfc7e0;font-size:12.5px}
.vc-logo{display:block;filter:drop-shadow(0 0 8px rgba(20,184,166,.7))}
.vc-hero{display:flex;flex-direction:column;align-items:center;text-align:center;gap:18px;max-width:420px;width:100%}
.vc-hero h1{font-size:22px;font-weight:700;margin:0}.vc-hero p{font-size:13.5px;color:#958da1;margin:0;line-height:1.9}
.vc-bar{width:100%;height:5px;border-radius:99px;background:#1b1b23;overflow:hidden}.vc-bar i{display:block;height:100%;width:8%;border-radius:99px;background:linear-gradient(90deg,#14b8a6,#38bdf8,#f5b83d);transition:width .4s ease;box-shadow:0 0 14px rgba(20,184,166,.8)}
.vc-steps{width:100%;display:flex;flex-direction:column;gap:9px;text-align:right}
.vc-st{display:flex;align-items:center;gap:10px;font-size:13px;color:#6f6880;transition:color .3s}
.vc-st.on{color:#e4e1ed}.vc-st.done{color:#f5b83d}
.vc-dot{width:16px;height:16px;border-radius:50%;border:2px solid #34343d;flex:none;position:relative}
.vc-st.on .vc-dot{border-color:#14b8a6;border-top-color:transparent;animation:vcsp .8s linear infinite}
.vc-st.done .vc-dot{border-color:#f5b83d;background:#f5b83d;animation:none}.vc-st.done .vc-dot:after{content:'';position:absolute;left:3px;top:0;width:4px;height:8px;border:solid #04100f;border-width:0 2px 2px 0;transform:rotate(45deg)}
@keyframes vcsp{to{transform:rotate(360deg)}}
.vc-err{color:#ffb4ab;font-size:13px}.vc-btn{background:#14b8a6;color:#fff;border:0;border-radius:12px;padding:10px 22px;font:inherit;font-weight:700;cursor:pointer}
.vc-rays{animation:vcrot 18s linear infinite;transform-origin:center}@keyframes vcrot{to{transform:rotate(360deg)}}
`;
  const st = document.createElement('style'); st.textContent = css; document.head.appendChild(st);

  const logo = (size = 40) => `<svg class="vc-logo" width="${size}" height="${size}" viewBox="0 0 64 64" aria-hidden="true">
<defs><radialGradient id="vcg" cx="50%" cy="50%" r="50%"><stop offset="0" stop-color="#000"/><stop offset=".6" stop-color="#04100f"/><stop offset="1" stop-color="#0a2e2b"/></radialGradient>
<linearGradient id="vcr" x1="0" x2="1"><stop offset="0" stop-color="#14b8a6"/><stop offset=".55" stop-color="#38bdf8"/><stop offset="1" stop-color="#f5b83d"/></linearGradient>
<filter id="vcb" x="-20%" y="-60%" width="140%" height="220%"><feGaussianBlur stdDeviation="2.2"/></filter></defs>
<g transform="rotate(-22 32 32)">
<ellipse cx="32" cy="32" rx="29" ry="9.5" fill="none" stroke="url(#vcr)" stroke-width="4" opacity=".55" filter="url(#vcb)"/>
<ellipse cx="32" cy="32" rx="29" ry="9.5" fill="none" stroke="url(#vcr)" stroke-width="1.6" opacity=".9"/>
<circle cx="32" cy="32" r="14.5" fill="url(#vcg)" stroke="#14b8a6" stroke-opacity=".6" stroke-width="1"/>
<path d="M3 32A29 9.5 0 0 0 61 32" fill="none" stroke="url(#vcr)" stroke-width="2.8" stroke-linecap="round"/>
<circle r="2.3" fill="#f5b83d"><animateMotion dur="3.2s" repeatCount="indefinite" path="M3 32A29 9.5 0 1 1 61 32A29 9.5 0 1 1 3 32"/></circle>
</g></svg>`;
  const word = '<span class="vc-word"><b>Void</b>Captcha</span>';
  const badge = () => `<span class="vc-badge">${logo(20)}<span>محافظت‌شده توسط</span>${word}</span>`;

  const $ = (h) => { const d = document.createElement('div'); d.innerHTML = h.trim(); return d.firstChild; };
  const call = async (path, body) => {
    const r = await fetch('/api/vc/' + path, body === undefined ? { credentials: 'same-origin' } : { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    const j = await r.json().catch(() => ({})); j._s = r.status; return j;
  };
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

  /* ---------- اثبات کار (SHA-256 همزمان؛ داخل Worker) ---------- */
  function powMain() {
    const P = []; for (let n = 2; P.length < 64; n++) { let ok = true; for (let d = 2; d * d <= n; d++) if (n % d === 0) { ok = false; break; } if (ok) P.push(n); }
    const frac = (x) => Math.floor((x - Math.floor(x)) * 4294967296) >>> 0;
    const K = P.map((p) => frac(Math.cbrt(p))), H0 = P.slice(0, 8).map((p) => frac(Math.sqrt(p)));
    const w = new Int32Array(64), h = new Int32Array(8);
    const rr = (x, n) => (x >>> n) | (x << (32 - n));
    function sha(bytes) {
      const l = bytes.length, total = ((l + 9 + 63) >> 6) << 6, buf = new Uint8Array(total);
      buf.set(bytes); buf[l] = 0x80; const dv = new DataView(buf.buffer); dv.setUint32(total - 4, l * 8);
      h.set(H0);
      for (let o = 0; o < total; o += 64) {
        for (let i = 0; i < 16; i++) w[i] = dv.getInt32(o + i * 4);
        for (let i = 16; i < 64; i++) { const a = w[i - 15], b = w[i - 2]; w[i] = (w[i - 16] + (rr(a, 7) ^ rr(a, 18) ^ (a >>> 3)) + w[i - 7] + (rr(b, 17) ^ rr(b, 19) ^ (b >>> 10))) | 0; }
        let [a, b, c, d, e, f, g, hh] = h;
        for (let i = 0; i < 64; i++) {
          const t1 = (hh + (rr(e, 6) ^ rr(e, 11) ^ rr(e, 25)) + ((e & f) ^ (~e & g)) + K[i] + w[i]) | 0;
          const t2 = ((rr(a, 2) ^ rr(a, 13) ^ rr(a, 22)) + ((a & b) ^ (a & c) ^ (b & c))) | 0;
          hh = g; g = f; f = e; e = (d + t1) | 0; d = c; c = b; b = a; a = (t1 + t2) | 0;
        }
        h[0] += a; h[1] += b; h[2] += c; h[3] += d; h[4] += e; h[5] += f; h[6] += g; h[7] += hh;
      }
      return h;
    }
    const zeros = (hw) => { let n = 0; for (let i = 0; i < 8; i++) { const v = hw[i] >>> 0; if (v === 0) { n += 32; continue; } return n + Math.clz32(v); } return n; };
    const enc = new TextEncoder();
    function solve(seed, bits) {
      for (let nonce = 0; ; nonce++) if (zeros(sha(enc.encode(seed + ':' + nonce))) >= bits) return nonce;
    }
    return { solve, sha, zeros };
  }
  const powSolve = (seed, bits) => new Promise((resolve) => {
    const run = () => resolve(powMain().solve(seed, bits));
    try {
      const url = URL.createObjectURL(new Blob([`const m=(${powMain.toString()})();onmessage=e=>postMessage(m.solve(e.data.seed,e.data.bits))`], { type: 'text/javascript' }));
      const wk = new Worker(url);
      wk.onmessage = (e) => { wk.terminate(); URL.revokeObjectURL(url); resolve(e.data); };
      wk.onerror = () => { wk.terminate(); setTimeout(run, 0); };
      wk.postMessage({ seed, bits });
    } catch { setTimeout(run, 0); }
  });

  /* ---------- «در حال بررسی مرورگر شما…» ---------- */
  async function ensureBrowser(force) {
    let ch = await call('browser');
    if (ch.ok && ch.valid && !force) return true;
    if (!ch.ok || ch.valid) { ch = await call('browser'); if (ch.valid) return true; }
    const ov = $(`<div class="vc-ov vc-full"><div class="vc-hero">${logo(86)}
      <div>${word.replace('class="vc-word"', 'class="vc-word" style="font-size:26px"')}</div>
      <h1>در حال بررسی مرورگر شما…</h1>
      <p>برای دسترسی به شبکه مرکزی آوند، مرورگر شما بررسی می‌شود. چند ثانیه صبر کنید.</p>
      <div class="vc-bar"><i></i></div>
      <div class="vc-steps"><div class="vc-st"><span class="vc-dot"></span>بررسی محیط مرورگر</div><div class="vc-st"><span class="vc-dot"></span>تأیید اجرای اسکریپت‌ها</div><div class="vc-st"><span class="vc-dot"></span>محاسبه‌ی اثبات کار</div></div>
      <div class="vc-err" hidden></div>${badge()}</div></div>`);
    document.documentElement.appendChild(ov);
    const steps = [...ov.querySelectorAll('.vc-st')], bar = ov.querySelector('.vc-bar i'), err = ov.querySelector('.vc-err');
    const go = async (i, pct, fn) => { steps[i].classList.add('on'); bar.style.width = pct + '%'; const r = fn ? await fn() : await sleep(450); steps[i].classList.remove('on'); steps[i].classList.add('done'); return r; };
    const attempt = async (c) => {
      err.hidden = true; steps.forEach((s) => s.className = 'vc-st'); bar.style.width = '8%';
      const t0 = Date.now();
      const sig = await go(0, 30, async () => { await sleep(500); return { wd: navigator.webdriver === true, lang: (navigator.languages || []).length, tz: Intl.DateTimeFormat().resolvedOptions().timeZone }; });
      await go(1, 55, async () => { await sleep(420); });
      const nonce = await go(2, 90, () => powSolve(c.seed, c.bits));
      const r = await call('browser', { id: c.id, nonce: String(nonce), sig });
      if (!r.ok) throw new Error(r.error || 'بررسی مرورگر ناموفق بود.');
      bar.style.width = '100%'; await sleep(Math.max(0, 1500 - (Date.now() - t0)));
    };
    for (;;) {
      try { await attempt(ch); break; }
      catch (e) {
        err.textContent = e.message; err.hidden = false;
        await new Promise((ok) => { const b = $('<button class="vc-btn">تلاش دوباره</button>'); err.after(b); b.onclick = () => { b.remove(); ok(); }; });
        ch = await call('browser'); if (ch.valid) break;
      }
    }
    ov.classList.add('vc-out'); setTimeout(() => ov.remove(), 500);
    return true;
  }

  /* ---------- پازل ---------- */
  let cache = null;
  const reset = () => { cache = null; };
  function solve() {
    if (cache && Date.now() < cache.exp && cache.uses < 3) { cache.uses++; return Promise.resolve(cache.token); }
    return new Promise((resolve, reject) => {
      const ov = $(`<div class="vc-ov" role="dialog" aria-modal="true"><div class="vc-card">
        <div class="vc-top">${logo(38)}<div><small class="vc-pre">برای دسترسی به شبکه مرکزی آوند:</small><b>دروازه کپچا:</b><small>با کشیدن فلش، پازل را درست کن</small></div><button class="vc-ref" aria-label="پازل جدید" title="پازل جدید">↻</button><button class="vc-x" aria-label="بستن">×</button></div>
        <div class="vc-stage"><img class="vc-bg" alt=""/><img class="vc-piece" alt=""/><div class="vc-msg"></div></div>
        <div class="vc-track"><div class="vc-fill"></div><span class="vc-hint">فلش را بکش و قطعه را جا بینداز</span><div class="vc-handle"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12h15M13 6l6 6-6 6"/></svg></div></div>
        <div class="vc-foot">${badge()}</div></div></div>`);
      document.documentElement.appendChild(ov);
      const q = (s) => ov.querySelector(s);
      const bg = q('.vc-bg'), piece = q('.vc-piece'), msg = q('.vc-msg'), handle = q('.vc-handle'), fill = q('.vc-fill'), hint = q('.vc-hint'), track = q('.vc-track');
      let pz = null, busy = true, drag = null;
      const close = () => ov.remove();
      const say = (t, cls) => { msg.textContent = t; msg.className = 'vc-msg on ' + cls; };
      const place = (p) => { handle.style.left = p + 'px'; fill.style.width = p + 22 + 'px'; if (pz) { piece.style.left = p - 5 + 'px'; } };
      async function load() {
        busy = true; msg.className = 'vc-msg'; place(0); hint.style.opacity = 1; piece.style.visibility = 'hidden';
        const r = await call('puzzle');
        if (!r.ok) { say(r._s === 429 ? 'تلاش‌ها زیاد بود؛ کمی بعد دوباره امتحان کنید.' : (r.error || 'خطا در دریافت پازل'), 'vc-bad'); return; }
        pz = r; bg.src = r.bg; piece.src = r.piece; piece.style.top = r.ty - 8 + 'px'; piece.style.visibility = 'visible'; place(0); busy = false;
      }
      handle.addEventListener('pointerdown', (e) => {
        if (busy || !pz) return; e.preventDefault(); handle.setPointerCapture(e.pointerId);
        drag = { x0: e.clientX, y0: e.clientY, t0: performance.now(), track: [], last: 0 }; hint.style.opacity = 0;
      });
      handle.addEventListener('pointermove', (e) => {
        if (!drag) return;
        const p = Math.max(0, Math.min(track.clientWidth - 44, e.clientX - drag.x0)); place(p);
        const t = Math.round(performance.now() - drag.t0);
        if (drag.track.length < 380 && (t - drag.last >= 8 || !drag.track.length)) { drag.track.push([t, Math.round(p * 10) / 10, Math.round((e.clientY - drag.y0) * 10) / 10]); drag.last = t; }
        drag.p = p;
      });
      const end = async () => {
        if (!drag) return; const d = drag; drag = null; if (!d.track.length || busy) return; busy = true;
        const r = await call('puzzle', { id: pz.id, x: d.p, track: d.track, dur: Math.round(performance.now() - d.t0) });
        if (r.ok) { cache = { token: r.token, exp: Date.now() + 270000, uses: 1 }; say('✓ تأیید شد', 'vc-ok'); await sleep(650); close(); resolve(r.token); return; }
        track.classList.add('vc-shake'); say(r._s === 429 ? 'تلاش‌ها زیاد بود؛ کمی بعد دوباره امتحان کنید.' : 'تأیید نشد، دوباره تلاش کنید', 'vc-bad');
        await sleep(r._s === 429 ? 2500 : 900); track.classList.remove('vc-shake'); load();
      };
      handle.addEventListener('dragstart', (e) => e.preventDefault()); handle.addEventListener('pointerup', end); handle.addEventListener('pointercancel', end);
      q('.vc-ref').onclick = () => { if (!drag) load(); };
      q('.vc-x').onclick = () => { close(); reject(new Error('cancel')); };
      load();
    });
  }

  function mountBadges() { document.querySelectorAll('[data-vc-badge]').forEach((el) => { if (!el.firstChild) el.innerHTML = badge(); }); }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mountBadges); else mountBadges();

  window.VC = { logo, word, badge, solve, reset, ensureBrowser, mountBadges, _pow: powMain };
})();
