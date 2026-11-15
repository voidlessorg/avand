(() => {
  const S = { me: null, files: [], folders: [], view: 'all', folder: '', q: '', sel: new Set(), mode: 'grid', sort: 'taken' };
  try { S.mode = localStorage.getItem('avand_mode') || 'grid'; } catch {}
  const NAV = [['all', 'home', 'همه‌ی فایل‌ها'], ['image', 'image', 'عکس‌ها'], ['video', 'video', 'ویدیوها و فیلم‌ها'], ['doc', 'doc', 'اسناد و فایل‌های اداری']];
  const KIC = { image: 'image', video: 'video', doc: 'doc', audio: 'music', other: 'file' };
  const fileTs = (f) => f.taken || f.created;

  async function load() {
    const [m, fl, fo] = await Promise.all([api('me'), api('files'), api('folders')]);
    S.me = m; S.files = fl.files; S.folders = fo.folders; renderAll();
  }
  const visible = () => S.files.filter((f) => {
    if (S.view === 'folder') { if (f.folder !== S.folder) return false; }
    else if (S.view !== 'all' && f.kind !== S.view) return false;
    return !S.q || f.name.toLowerCase().includes(S.q);
  });
  const count = (k) => S.files.filter((f) => k === 'all' || f.kind === k).length;

  function renderSide() {
    $('#brand').innerHTML = avandLogo(40) + '<div><b>آوند</b><small>AVAND</small></div>';
    $('#nav').innerHTML = NAV.map(([k, i, t]) => `<a href="#" data-v="${k}" class="${S.view === k ? 'on' : ''}">${ic(i)}<span>${t}</span><span class="cnt num">${fa(count(k))}</span></a>`).join('') +
      (S.me.admin ? `<a href="/admin">${ic('settings')}<span>پنل مدیریت</span></a>` : '');
    $('#folders').innerHTML = S.folders.map((f) => `<a href="#" data-f="${f.id}" class="${S.view === 'folder' && S.folder === f.id ? 'on' : ''}">${ic('folder')}<span>${esc(f.name)}</span><span class="cnt num">${fa(f.count)}</span></a>`).join('') || '<div class="muted" style="font-size:12.5px;padding:2px 12px">هنوز پوشه‌ای ندارید.</div>';
    const q = S.me.quota, u = S.me.used, pct = q ? Math.min(100, u / q * 100) : 0;
    $('#quota').innerHTML = `<div class="row" style="font-size:12.5px;color:var(--muted);margin-bottom:6px">${ic('drive')}<span class="grow">فضای مصرفی</span><span class="num">${fmtSize(u)}${q ? ' از ' + fmtSize(q) : ''}</span></div>` + (q ? `<div class="meter ${pct > 85 ? 'warn' : ''}"><i style="width:${pct}%"></i></div>` : '<div class="muted" style="font-size:12px">سهمیه‌ی نامحدود</div>');
    $('#supportBox').innerHTML = supportCard();
    $$('[data-v]').forEach((a) => a.addEventListener('click', (e) => { e.preventDefault(); S.view = a.dataset.v; S.folder = ''; S.sel.clear(); $('#side').classList.remove('open'); renderAll(); }));
    $$('[data-f]').forEach((a) => {
      a.addEventListener('click', (e) => { e.preventDefault(); S.view = 'folder'; S.folder = a.dataset.f; S.sel.clear(); $('#side').classList.remove('open'); renderAll(); });
      a.addEventListener('contextmenu', (e) => { e.preventDefault(); folderMenu(a.dataset.f); });
    });
  }

  function tile(f) {
    const on = S.sel.has(f.id) ? ' on' : '';
    const media = f.kind === 'image' || f.kind === 'video' ? `<img loading="lazy" src="/t/${f.id}" alt="">` : `<div class="ph">${ic(KIC[f.kind] || 'file')}</div>`;
    return `<div class="tile${on}${f.kind === 'video' ? ' wide' : ''}" data-id="${f.id}">${media}${f.kind === 'video' ? `<span class="badge">${ic('play')}</span>` : ''}<span class="sel">${ic('check')}</span><div class="cap">${esc(f.name)}</div></div>`;
  }
  function li(f) {
    return `<div class="li${S.sel.has(f.id) ? ' on' : ''}" data-id="${f.id}"><div class="fi">${ic(KIC[f.kind] || 'file')}</div><div class="grow"><div class="nm">${esc(f.name)}</div><div class="meta">${fmtSize(f.size)} · ${dateFa(fileTs(f))}</div></div><button class="btn ghost sm icon" data-more="${f.id}" aria-label="گزینه‌ها">${ic('more')}</button></div>`;
  }
  function renderContent() {
    const list = visible(); const title = S.view === 'folder' ? (S.folders.find((f) => f.id === S.folder) || {}).name || 'پوشه' : NAV.find((n) => n[0] === S.view)[2];
    const n = S.sel.size;
    let h = `<div class="toolbar"><h1>${esc(title)}</h1><span class="chip num">${fa(list.length)} مورد</span><span class="grow"></span>`;
    if (n) h += `<span class="chip pri num">${fa(n)} انتخاب‌شده</span><button class="btn sm" id="bDl">${ic('download')}دانلود</button><button class="btn sm" id="bMv">${ic('move')}انتقال</button><button class="btn sm danger" id="bDel">${ic('trash')}حذف</button><button class="btn ghost sm" id="bClr">لغو</button>`;
    else h += `<select class="input" id="sortSel" style="width:auto;padding:6px 12px 6px 38px"><option value="taken">جدیدترین</option><option value="name">نام</option><option value="size">حجم</option></select>`;
    h += '</div>';
    if (!list.length) {
      h += `<div class="empty">${ic('upload')}<b style="color:var(--text);font-size:17px">${S.q ? 'چیزی پیدا نشد' : 'اینجا هنوز خالی است'}</b><span>${S.q ? 'عبارت دیگری را امتحان کنید.' : 'فایل‌ها را بکشید و اینجا رها کنید، یا دکمه‌ی «آپلود» را بزنید.'}</span>${S.q ? '' : `<button class="btn pri" id="emptyUp">${ic('upload')} آپلود فایل</button>`}</div>`;
    } else if (S.mode === 'list' || S.view === 'doc') {
      h += `<div class="list">${list.map(li).join('')}</div>`;
    } else {
      const sorted = S.sort === 'taken';
      if (sorted) {
        const g = new Map();
        list.forEach((f) => { const k = dateFa(fileTs(f), true); (g.get(k) || g.set(k, []).get(k)).push(f); });
        g.forEach((arr, k) => { h += `<div class="day">${k}</div><div class="grid sq${S.sel.size ? ' selmode' : ''}">${arr.map(tile).join('')}</div>`; });
      } else h += `<div class="grid sq${S.sel.size ? ' selmode' : ''}">${list.map(tile).join('')}</div>`;
    }
    $('#content').innerHTML = h;
    const ss = $('#sortSel'); if (ss) { ss.value = S.sort; ss.addEventListener('change', async () => { S.sort = ss.value; const r = await api('files?sort=' + S.sort); S.files = r.files; renderContent(); }); }
    $$('.tile,.li').forEach((el) => {
      el.addEventListener('click', (e) => {
        const id = el.dataset.id;
        if (e.target.closest('[data-more]')) return fileMenu(id);
        if (e.target.closest('.sel') || S.sel.size || e.ctrlKey || e.metaKey) { S.sel.has(id) ? S.sel.delete(id) : S.sel.add(id); renderContent(); return; }
        openViewer(id);
      });
      el.addEventListener('contextmenu', (e) => { e.preventDefault(); fileMenu(el.dataset.id); });
    });
    const on = (id, fn) => { const b = $(id); if (b) b.addEventListener('click', fn); };
    on('#emptyUp', () => $('#pick').click()); on('#bClr', () => { S.sel.clear(); renderContent(); });
    on('#bDl', () => [...S.sel].forEach((id, i) => setTimeout(() => dl(id), i * 350)));
    on('#bDel', async () => { if (await confirmBox(`${fa(S.sel.size)} فایل برای همیشه حذف شود؟`, 'حذف', true)) { await api('files/delete', { body: { ids: [...S.sel] } }); S.sel.clear(); toast('حذف شد', 'ok'); load(); } });
    on('#bMv', () => moveDlg([...S.sel]));
  }
  const dl = (id) => { const f = S.files.find((x) => x.id === id); const a = document.createElement('a'); a.href = `/f/${id}?dl=1`; a.download = f ? f.name : ''; document.body.appendChild(a); a.click(); a.remove(); };

  function fileMenu(id) {
    const f = S.files.find((x) => x.id === id); if (!f) return;
    const m = modal(`<h2 style="overflow:hidden;text-overflow:ellipsis;white-space:nowrap">${esc(f.name)}</h2><div class="muted" style="margin:-8px 0 14px;font-size:13px">${fmtSize(f.size)} · ${timeFa(f.created)}</div><div class="col">
      <button class="btn" data-a="open">${ic('play')}باز کردن</button><button class="btn" data-a="dl">${ic('download')}دانلود</button><button class="btn" data-a="ren">${ic('edit')}تغییر نام</button><button class="btn" data-a="mv">${ic('move')}انتقال به پوشه</button><button class="btn danger" data-a="del">${ic('trash')}حذف</button></div>`);
    $$('[data-a]', m).forEach((b) => b.addEventListener('click', async () => {
      const a = b.dataset.a; m.close();
      if (a === 'open') openViewer(id); else if (a === 'dl') dl(id); else if (a === 'mv') moveDlg([id]);
      else if (a === 'ren') { const nn = await ask('نام جدید', f.name); if (nn) { try { await api('files/rename', { body: { id, name: nn } }); load(); } catch (e) { toast(e.message, 'bad'); } } }
      else if (a === 'del' && await confirmBox(`«${f.name}» حذف شود؟`, 'حذف', true)) { await api('files/delete', { body: { ids: [id] } }); load(); }
    }));
  }
  const ask = (title, val = '') => new Promise((res) => {
    const m = modal(`<h2>${esc(title)}</h2><form id="af"><input class="input" id="av" value="${esc(val)}" maxlength="120" required><div class="row" style="margin-top:14px"><button type="button" class="btn grow" data-close>انصراف</button><button class="btn pri grow">تأیید</button></div></form>`);
    const i = $('#av', m); i.focus(); i.select();
    $('#af', m).addEventListener('submit', (e) => { e.preventDefault(); const v = i.value.trim(); m.close(); res(v); });
    const c = m.close; m.close = () => { c(); res(null); };
  });
  function moveDlg(ids) {
    const m = modal(`<h2>انتقال به…</h2><div class="col"><button class="btn" data-fo="">${ic('home')}بدون پوشه</button>${S.folders.map((f) => `<button class="btn" data-fo="${f.id}">${ic('folder')}${esc(f.name)}</button>`).join('')}</div>`);
    $$('[data-fo]', m).forEach((b) => b.addEventListener('click', async () => { await api('files/move', { body: { ids, folder: b.dataset.fo } }); m.close(); S.sel.clear(); load(); }));
  }
  function folderMenu(id) {
    const f = S.folders.find((x) => x.id === id); if (!f) return;
    const m = modal(`<h2>${esc(f.name)}</h2><div class="col"><button class="btn" data-a="r">${ic('edit')}تغییر نام</button><button class="btn danger" data-a="d">${ic('trash')}حذف پوشه (فایل‌ها می‌مانند)</button></div>`);
    $$('[data-a]', m).forEach((b) => b.addEventListener('click', async () => {
      m.close();
      if (b.dataset.a === 'r') { const n = await ask('نام پوشه', f.name); if (n) { await api('folders', { body: { id, name: n } }); load(); } }
      else if (await confirmBox('پوشه حذف شود؟ فایل‌های داخل آن حذف نمی‌شوند.', 'حذف', true)) { await api('folders/delete', { body: { id } }); if (S.folder === id) { S.view = 'all'; S.folder = ''; } load(); }
    }));
  }

  /* نمایشگر */
  function openViewer(id) {
    const list = visible(); let i = list.findIndex((f) => f.id === id); if (i < 0) return;
    const lb = document.createElement('div'); lb.className = 'lightbox';
    const draw = () => {
      const f = list[i]; let body;
      if (f.kind === 'image') body = `<img src="/f/${f.id}" alt="${esc(f.name)}">`;
      else if (f.kind === 'video') body = `<video src="/f/${f.id}" controls autoplay playsinline></video>`;
      else if (f.kind === 'audio') body = `<audio src="/f/${f.id}" controls autoplay style="width:min(560px,90%)"></audio>`;
      else if (f.mime === 'application/pdf') body = `<iframe src="/f/${f.id}" title="${esc(f.name)}"></iframe>`;
      else if (/^text\//.test(f.mime) && f.size < 2e6) body = '<pre>در حال بارگذاری…</pre>';
      else body = `<div class="col" style="align-items:center;color:#fff;text-align:center"><div style="font-size:64px">${ic(KIC[f.kind] || 'file')}</div><div>پیش‌نمایش این نوع فایل ممکن نیست.</div><button class="btn pri" id="lbDl2">${ic('download')}دانلود</button></div>`;
      lb.innerHTML = `<div class="bar"><span class="nm">${esc(f.name)}</span><span class="muted num" style="color:#9ab">${fmtSize(f.size)} · ${fa(i + 1)}/${fa(list.length)}</span><button class="btn sm" id="lbDl">${ic('download')}دانلود</button><button class="btn sm" id="lbX" aria-label="بستن">${ic('x')}</button></div><div class="stage">${list.length > 1 ? `<button class="nav-btn prev" id="lbP" aria-label="قبلی">${ic('chevR')}</button><button class="nav-btn next" id="lbN" aria-label="بعدی">${ic('chevL')}</button>` : ''}${body}</div>`;
      $('#lbX', lb).onclick = close; $('#lbDl', lb).onclick = () => dl(f.id);
      const d2 = $('#lbDl2', lb); if (d2) d2.onclick = () => dl(f.id);
      const p = $('#lbP', lb), n = $('#lbN', lb); if (p) p.onclick = () => go(-1); if (n) n.onclick = () => go(1);
      const pre = $('pre', lb); if (pre) fetch('/f/' + f.id, { credentials: 'same-origin' }).then((r) => r.text()).then((t) => { pre.textContent = t; });
    };
    const go = (d) => { i = (i + d + list.length) % list.length; draw(); };
    const key = (e) => { if (e.key === 'Escape') close(); else if (e.key === 'ArrowRight') go(-1); else if (e.key === 'ArrowLeft') go(1); };
    const close = () => { lb.remove(); document.removeEventListener('keydown', key); };
    document.addEventListener('keydown', key); document.body.appendChild(lb); draw();
  }

  /* آپلود */
  const queue = []; let running = 0;
  function enqueue(files) {
    for (const file of files) { if (!file.size && !file.type) continue; const it = { file, el: null, st: 'wait' }; queue.push(it); }
    $('#tray').classList.remove('hide'); pump(); trayDraw();
  }
  function trayDraw() {
    const wait = queue.filter((x) => x.st === 'wait' || x.st === 'up').length;
    $('#trayT').textContent = wait ? `در حال آپلود ${fa(wait)} فایل…` : 'آپلود کامل شد';
    $('#items').innerHTML = queue.slice(-40).map((x, i) => `<div class="up ${x.st === 'err' ? 'err' : x.st === 'done' ? 'done' : ''}"><div class="nm"><span>${esc(x.file.name)}</span><span class="num">${x.st === 'err' ? esc(x.msg || 'خطا') : x.st === 'done' ? '✓' : fa(Math.round((x.p || 0) * 100)) + '٪'}</span></div><div class="meter"><i style="width:${Math.round((x.p || 0) * 100)}%"></i></div></div>`).join('');
  }
  function pump() {
    while (running < 2) {
      const it = queue.find((x) => x.st === 'wait'); if (!it) break;
      it.st = 'up'; running++;
      const x = new XMLHttpRequest(); const f = it.file;
      const qs = new URLSearchParams({ name: f.name, mtime: String(Math.floor((f.lastModified || Date.now()) / 1000)) }); if (S.view === 'folder') qs.set('folder', S.folder);
      x.open('POST', '/api/upload?' + qs); x.withCredentials = true;
      x.upload.onprogress = (e) => { if (e.lengthComputable) { it.p = e.loaded / e.total; trayDraw(); } };
      x.onload = () => { let j = {}; try { j = JSON.parse(x.responseText); } catch {} if (j.ok) { it.st = 'done'; it.p = 1; } else { it.st = 'err'; it.msg = j.error || 'خطا'; } fin(); };
      x.onerror = () => { it.st = 'err'; it.msg = 'قطع ارتباط'; fin(); };
      x.send(f);
    }
  }
  const fin = () => { running--; trayDraw(); if (!queue.some((x) => x.st === 'wait' || x.st === 'up')) { load(); setTimeout(() => { if (!queue.some((x) => x.st === 'up' || x.st === 'wait') && !queue.some((x) => x.st === 'err')) $('#tray').classList.add('hide'); }, 3500); } else pump(); };

  /* تنظیمات */
  function settings() {
    const m = modal(`<h2>تنظیمات</h2><div class="card flat" style="margin-bottom:12px"><div class="row"><div class="grow"><b>${esc(S.me.username)}</b><div class="muted" style="font-size:13px">${S.me.admin ? 'مدیر' : 'عضو'} · ${fa(S.me.count)} فایل</div></div><button class="btn sm" id="stTheme">${ic('sun')}تم</button></div></div>
<div class="card flat" style="margin-bottom:12px"><b>آدرس دسترسی از دستگاه‌های دیگر</b><div class="col" style="gap:4px;margin-top:8px">${(S.me.urls || []).map((u) => `<span class="ltr">${esc(u)}</span>`).join('') || '<span class="muted">—</span>'}</div></div>
<form id="pwf" class="card flat" style="margin-bottom:12px" autocomplete="on"><b>تغییر رمز عبور</b><input type="text" name="username" value="${esc(S.me.username)}" autocomplete="username" hidden>
<label class="field" style="margin-top:10px"><span>رمز فعلی</span><input class="input ltr" type="password" name="current" autocomplete="current-password" required></label>
<label class="field"><span>رمز جدید</span><input class="input ltr" type="password" name="new" autocomplete="new-password" minlength="8" required></label><button class="btn pri">ذخیره</button></form>
<div class="row wrap"><button class="btn" id="stTour">${ic('help')}تور آموزشی</button><button class="btn danger" id="stOut">${ic('logout')}خروج از حساب</button>${S.me.admin ? `<button class="btn danger" id="stQuit">${ic('x')}بستن برنامه‌ی آوند</button>` : ''}<button class="btn grow" data-close>بستن</button></div>`);
    $('#stTheme', m).onclick = () => theme.toggle();
    $('#stTour', m).onclick = () => { m.close(); tour(true); };
    $('#stOut', m).onclick = async () => { await api('logout', { body: {} }); location.href = '/'; };
    const q = $('#stQuit', m); if (q) q.onclick = async () => { if (!(await confirmBox('آوند کاملاً بسته شود؟ تا اجرای دوباره، دستگاه‌های دیگر به فایل‌ها دسترسی ندارند.', 'بستن', true))) return; try { await api('admin/quit', { body: {} }); document.body.innerHTML = '<div class="auth"><div class="hero">' + avandLogo(64) + '<h1>آوند بسته شد</h1><p>برای اجرای دوباره، از میان‌بر آوند استفاده کنید.</p></div></div>'; } catch (e) { toast(e.message, 'bad'); } };
    $('#pwf', m).addEventListener('submit', async (e) => { e.preventDefault(); const f = e.target; try { await api('password', { body: { current: f.current.value, new: f.new.value } }); toast('رمز تغییر کرد', 'ok'); m.close(); } catch (er) { toast(er.message, 'bad'); } });
  }

  /* تور آموزشی دستگاه */
  function tour(force) {
    let seen = false; try { seen = localStorage.getItem('avand_tour_v1') === '1'; } catch {}
    if (seen && !force) return;
    const steps = [
      ['', 'به آوند خوش آمدید', 'این یک مرکز فایل خصوصی روی شبکه‌ی محلی است. در ۶ قدم کوتاه کار با آن را یاد می‌گیرید. (فقط یک‌بار برای هر دستگاه نمایش داده می‌شود.)'],
      ['#upBtn', 'آپلود فایل', 'با این دکمه عکس، فیلم و اسناد را بفرستید. می‌توانید فایل‌ها را مستقیم روی صفحه بکشید و رها کنید.'],
      ['#nav', 'دسته‌بندی خودکار', 'فایل‌ها خودشان به عکس، ویدیو و اسناد اداری تقسیم می‌شوند.'],
      ['#newFolder', 'پوشه‌ها', 'برای مرتب‌سازی پوشه بسازید. روی نام پوشه کلیک راست (یا نگه‌داشتن) کنید تا تغییر نام یا حذف شود.'],
      ['#q', 'جست‌وجو', 'با نوشتن بخشی از نام فایل، سریع آن را پیدا کنید.'],
      ['#setBtn', 'تنظیمات و رمز عبور', 'رمز را تغییر دهید یا آدرس دسترسی از موبایل را ببینید. هنگام ورود، از مرورگر بخواهید رمز را ذخیره کند تا دفعه‌ی بعد خودکار وارد شوید.'],
      ['#supportBox', 'حمایت از آوند', 'آوند رایگان است. اگر برایتان مفید بود، از طریق daramet.com/zohalgame حمایت کنید؛ این بزرگ‌ترین کمک به ادامه‌ی پروژه است.'],
    ];
    let i = 0; const hole = document.createElement('div'); hole.className = 'tour-hole'; const tip = document.createElement('div'); tip.className = 'tour-tip';
    const mask = document.createElement('div'); mask.className = 'tour-mask'; document.body.append(mask, hole, tip);
    const done = () => { try { localStorage.setItem('avand_tour_v1', '1'); } catch {} mask.remove(); hole.remove(); tip.remove(); window.removeEventListener('resize', place); };
    function place() {
      const [sel, t, d] = steps[i]; const el = sel ? $(sel) : null;
      if (el && el.closest('.side') && matchMedia('(max-width:860px)').matches) $('#side').classList.add('open'); else if (sel && !el?.closest('.side')) $('#side').classList.remove('open');
      let r = el ? el.getBoundingClientRect() : { left: innerWidth / 2, top: innerHeight / 2, width: 0, height: 0, bottom: innerHeight / 2 };
      if (el) { const p = 6; hole.style.cssText = `left:${r.left - p}px;top:${r.top - p}px;width:${r.width + 2 * p}px;height:${r.height + 2 * p}px`; }
      else hole.style.cssText = `left:${innerWidth / 2}px;top:${innerHeight / 2}px;width:0;height:0`;
      tip.innerHTML = `<div class="dots">${steps.map((_, k) => `<i class="${k === i ? 'on' : ''}"></i>`).join('')}</div><h3>${ic('info')}${esc(t)}</h3><p>${esc(d)}</p><div class="row"><button class="btn ghost sm" id="tSkip">رد کردن</button><span class="grow"></span>${i ? '<button class="btn sm" id="tPrev">قبلی</button>' : ''}<button class="btn pri sm" id="tNext">${i === steps.length - 1 ? 'شروع' : 'بعدی'}</button></div>`;
      const tw = Math.min(320, innerWidth - 24), th = tip.offsetHeight || 190;
      let x = el ? r.left + r.width / 2 - tw / 2 : innerWidth / 2 - tw / 2, y = el ? r.bottom + 18 : innerHeight / 2 - th / 2;
      if (y + th > innerHeight - 10) y = Math.max(10, r.top - th - 18);
      x = Math.max(12, Math.min(innerWidth - tw - 12, x)); tip.style.left = x + 'px'; tip.style.top = Math.max(10, y) + 'px';
      $('#tSkip', tip).onclick = done; $('#tNext', tip).onclick = () => { if (i === steps.length - 1) done(); else { i++; place(); } };
      const pv = $('#tPrev', tip); if (pv) pv.onclick = () => { i--; place(); };
    }
    window.addEventListener('resize', place); place();
  }

  function renderAll() { renderSide(); renderContent(); }
  function init() {
    $('#newFolder').innerHTML = ic('folderPlus'); $('#burger').innerHTML = ic('menu'); $('#sIc').innerHTML = ic('search');
    $('#upBtn').innerHTML = ic('upload') + '<span>آپلود</span>'; $('#setBtn').innerHTML = ic('settings'); $('#trayX').innerHTML = ic('x');
    const vb = () => { $('#viewBtn').innerHTML = ic(S.mode === 'grid' ? 'list' : 'grid'); };
    vb();
    $('#viewBtn').onclick = () => { S.mode = S.mode === 'grid' ? 'list' : 'grid'; try { localStorage.setItem('avand_mode', S.mode); } catch {} vb(); renderContent(); };
    $('#burger').onclick = () => $('#side').classList.toggle('open');
    $('#upBtn').onclick = () => $('#pick').click();
    $('#pick').addEventListener('change', (e) => { enqueue([...e.target.files]); e.target.value = ''; });
    $('#setBtn').onclick = settings; $('#trayX').onclick = () => $('#tray').classList.add('hide');
    $('#newFolder').onclick = async () => { const n = await ask('نام پوشه‌ی جدید'); if (n) { try { await api('folders', { body: { name: n } }); load(); } catch (e) { toast(e.message, 'bad'); } } };
    let tm; $('#q').addEventListener('input', (e) => { clearTimeout(tm); tm = setTimeout(() => { S.q = e.target.value.trim().toLowerCase(); renderContent(); }, 150); });
    $('#dropT').innerHTML = ic('upload', '') + '<div>فایل‌ها را اینجا رها کنید</div>';
    let dc = 0;
    addEventListener('dragenter', (e) => { if ([...(e.dataTransfer?.types || [])].includes('Files')) { dc++; $('#drop').classList.add('on'); } });
    addEventListener('dragleave', () => { if (--dc <= 0) { dc = 0; $('#drop').classList.remove('on'); } });
    addEventListener('dragover', (e) => e.preventDefault());
    addEventListener('drop', (e) => { e.preventDefault(); dc = 0; $('#drop').classList.remove('on'); if (e.dataTransfer?.files?.length) enqueue([...e.dataTransfer.files]); });
    load().then(() => setTimeout(() => tour(false), 500)).catch(() => {});
  }
  init();
})();
