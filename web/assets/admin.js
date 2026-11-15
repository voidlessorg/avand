(() => {
  $('#brand').innerHTML = avandLogo(38) + '<div><b>مدیریت آوند</b><small>AVAND</small></div>';
  $('#th').innerHTML = ic('sun'); $('#th').onclick = () => theme.toggle();
  $('#back').innerHTML = ic('home') + 'فایل‌ها';
  $('#sb').innerHTML = ic('heart') + '<span>از آوند حمایت کنید — رایگان است و با حمایت شما ادامه پیدا می‌کند.</span><span class="ltr">daramet.com/zohalgame</span>';
  const GB = 1073741824;
  const qTxt = (q) => (q ? fmtSize(q) : 'نامحدود');
  const askGB = (t, def) => { const v = prompt(t, def); if (v === null) return null; const n = parseFloat(String(v).replace(/[۰-۹]/g, (d) => '۰۱۲۳۴۵۶۷۸۹'.indexOf(d))); return isNaN(n) || n < 0 ? null : Math.round(n * GB); };

  async function stats() {
    const s = await api('admin/stats');
    const d = (i, t, v) => `<div class="card stat"><small>${ic(i)}${t}</small><b class="num">${v}</b></div>`;
    $('#stats').innerHTML = d('users', 'کاربران', fa(s.users)) + d('file', 'فایل‌ها', fa(s.files)) + d('drive', 'حجم ذخیره‌شده', fmtSize(s.bytes)) + d('drive', 'فضای آزاد دیسک', fmtSize(s.disk_free));
  }
  async function users() {
    const r = await api('admin/users');
    $('#users').innerHTML = `<table class="t"><tr><th>نام کاربری</th><th>نقش</th><th>مصرف / سهمیه</th><th>فایل</th><th>وضعیت</th><th></th></tr>${r.users.map((u) => `<tr><td class="ltr">${esc(u.username)}</td><td>${u.admin ? 'مدیر' : 'عضو'}</td><td class="num">${fmtSize(u.used)} / ${qTxt(u.quota)}</td><td class="num">${fa(u.files)}</td><td><span class="chip ${u.active ? 'ok' : 'bad'}">${u.active ? 'فعال' : 'غیرفعال'}</span></td><td class="row" style="gap:6px;justify-content:flex-end"><button class="btn sm" data-q="${u.id}" data-qv="${u.quota}">سهمیه</button><button class="btn sm" data-r="${u.id}">رمز جدید</button><button class="btn sm" data-t="${u.id}" data-on="${u.active}">${u.active ? 'غیرفعال' : 'فعال'}</button><button class="btn sm danger" data-d="${u.id}" data-n="${esc(u.username)}">حذف</button></td></tr>`).join('')}</table>`;
    $$('[data-q]').forEach((b) => b.onclick = async () => { const q = askGB('سهمیه به گیگابایت (۰ = نامحدود):', (+b.dataset.qv / GB) || 0); if (q === null) return; try { await api('admin/user', { body: { id: b.dataset.q, quota: q } }); users(); } catch (e) { toast(e.message, 'bad'); } });
    $$('[data-r]').forEach((b) => b.onclick = async () => { const p = prompt('رمز جدید (حداقل ۸ نویسه):'); if (!p) return; try { await api('admin/user/reset', { body: { id: b.dataset.r, password: p } }); toast('رمز تغییر کرد', 'ok'); } catch (e) { toast(e.message, 'bad'); } });
    $$('[data-t]').forEach((b) => b.onclick = async () => { try { await api('admin/user', { body: { id: b.dataset.t, active: b.dataset.on !== 'true' } }); users(); } catch (e) { toast(e.message, 'bad'); } });
    $$('[data-d]').forEach((b) => b.onclick = async () => { if (await confirmBox(`کاربر «${b.dataset.n}» و همه‌ی فایل‌هایش برای همیشه حذف شود؟`, 'حذف', true)) { try { await api('admin/user/delete', { body: { id: b.dataset.d } }); users(); stats(); } catch (e) { toast(e.message, 'bad'); } } });
  }
  async function codes() {
    const r = await api('admin/codes');
    $('#codes').innerHTML = r.codes.length ? `<table class="t"><tr><th>کد</th><th>یادداشت</th><th>سهمیه</th><th>نقش</th><th>وضعیت</th><th></th></tr>${r.codes.map((c) => `<tr><td class="ltr num" style="font-weight:700">${esc(c.code)}</td><td>${esc(c.note || '—')}</td><td>${qTxt(c.quota)}</td><td>${c.admin ? 'مدیر' : 'عضو'}</td><td>${c.used_at ? `<span class="chip">استفاده شد (${esc(c.used_by_name || '—')})</span>` : '<span class="chip ok">آماده</span>'}</td><td style="text-align:left">${c.used_at ? '' : `<button class="btn sm" data-cp="${esc(c.code)}">${ic('copy')}</button> <button class="btn sm danger" data-cd="${esc(c.code)}">${ic('trash')}</button>`}</td></tr>`).join('')}</table>` : '<div class="empty" style="border:0">هنوز کدی نساخته‌اید.</div>';
    $$('[data-cp]').forEach((b) => b.onclick = () => { navigator.clipboard?.writeText(b.dataset.cp).then(() => toast('کپی شد', 'ok'), () => toast(b.dataset.cp)); });
    $$('[data-cd]').forEach((b) => b.onclick = async () => { try { await api('admin/codes/delete', { body: { code: b.dataset.cd } }); codes(); } catch (e) { toast(e.message, 'bad'); } });
  }
  $('#newCode').innerHTML = ic('plus') + 'ساخت کد جدید';
  $('#newCode').onclick = () => {
    const m = modal(`<h2>ساخت کد دعوت</h2><form id="cf"><label class="field"><span>تعداد</span><input class="input" name="count" type="number" min="1" max="100" value="1"></label><label class="field"><span>سهمیه (گیگابایت، ۰ = نامحدود)</span><input class="input" name="q" type="number" min="0" step="any" value="20"></label><label class="field"><span>یادداشت (مثلاً نام همکار)</span><input class="input" name="note" maxlength="100"></label><label class="check" style="margin-bottom:14px"><input type="checkbox" name="admin">دارای دسترسی مدیر</label><div class="row"><button type="button" class="btn grow" data-close>انصراف</button><button class="btn pri grow">بساز</button></div></form>`);
    $('#cf', m).addEventListener('submit', async (e) => { e.preventDefault(); const f = e.target; try { const r = await api('admin/codes', { body: { count: +f.count.value, quota: Math.round((+f.q.value || 0) * GB), note: f.note.value, admin: f.admin.checked } }); m.close(); codes(); modal(`<h2>کدهای ساخته‌شده</h2><pre class="ltr" style="user-select:all;background:var(--bg2);padding:12px;border-radius:12px;direction:ltr">${r.codes.map(esc).join('\n')}</pre><button class="btn pri" style="width:100%" data-close>باشه</button>`); } catch (er) { toast(er.message, 'bad'); } });
  };
  async function net() {
    const s = await api('admin/settings'); const u = s.update || {};
    $('#net').innerHTML = `<div class="row wrap" style="margin-bottom:12px"><div class="grow"><b>اتصال به شبکه‌ی مرکزی آوند</b><div class="muted" style="font-size:13px">روشن: کپچای ورود (VoidCaptcha) فعال و به‌روزرسانی خودکار از گیت‌هاب برقرار است. خاموش: حالت کاملاً لوکال، بدون کپچا و بدون هیچ ارتباط بیرونی.</div></div><button class="switch ${s.central ? 'on' : ''}" id="swC" aria-label="شبکه مرکزی"></button></div>
<div class="row wrap" style="margin-bottom:12px"><div class="grow"><b>به‌روزرسانی خودکار</b><div class="muted" style="font-size:13px">نیازمند اتصال به شبکه‌ی مرکزی. نسخه‌ها با امضای دیجیتال بررسی می‌شوند.</div></div><button class="switch ${s.auto_update && s.central ? 'on' : ''}" id="swU" ${s.central ? '' : 'disabled'} aria-label="آپدیت"></button></div>
<div class="row wrap"><span class="chip">نسخه ${esc(s.version)}</span><span class="chip ${s.central ? 'ok' : 'bad'}">${s.central ? 'متصل به شبکه‌ی مرکزی' : 'قطع دسترسی از شبکه‌ی مرکزی'}</span>${s.update_configured ? '' : '<span class="chip">مخزن به‌روزرسانی در این نسخه تنظیم نشده</span>'}${u.latest ? `<span class="chip pri">آخرین نسخه: ${esc(u.latest)}</span>` : ''}${u.status ? `<span class="chip">${esc(u.status)}</span>` : ''}${u.error ? `<span class="chip bad">${esc(u.error)}</span>` : ''}<span class="grow"></span><button class="btn" id="chk" ${s.central ? '' : 'disabled'}>${ic('refresh')}بررسی به‌روزرسانی</button></div>`;
    $('#swC').onclick = async () => { if (s.central && !(await confirmBox('با قطع شبکه‌ی مرکزی، کپچا خاموش می‌شود و دیگر به‌روزرسانی خودکار نمی‌گیرید. ادامه می‌دهید؟', 'قطع کنم', true))) return; await api('admin/settings', { body: { central: !s.central } }); net(); };
    $('#swU').onclick = async () => { await api('admin/settings', { body: { auto_update: !s.auto_update } }); net(); };
    $('#chk').onclick = async () => { try { await api('admin/update/check', { body: {} }); toast('در حال بررسی…'); setTimeout(net, 2500); } catch (e) { toast(e.message, 'bad'); } };
  }
  Promise.all([stats(), users(), codes(), net()]).catch((e) => toast(e.message, 'bad'));
})();
