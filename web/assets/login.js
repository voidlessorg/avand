(async () => {
  $('#lg').innerHTML = avandLogo(76); $('#sh').innerHTML = ic('heart') + ' از آوند حمایت کنید';
  $('#lan').innerHTML = ic('shield') + ' فقط روی شبکه‌ی محلی شما در دسترس است';
  let st = { setup: false, central: true };
  try { st = await api('state', { quiet: true }); } catch {}
  const qs = new URLSearchParams(location.search);
  let tab = st.setup ? 'setup' : (qs.get('t') === 'access' ? 'access' : 'member');
  const err = qs.get('e');
  if (err) history.replaceState(null, '', '/');
  const tabs = st.setup ? '' : `<div class="tabs"><button type="button" data-t="member" class="${tab === 'member' ? 'on' : ''}">ورود</button><button type="button" data-t="access" class="${tab === 'access' ? 'on' : ''}">عضویت با کد</button></div>`;
  const E = err ? `<div class="err">${esc(err)}</div>` : '';
  const pwNote = '<input type="hidden" name="captcha" value="">';
  function form() {
    if (tab === 'setup') return `<h2 style="margin:0 0 4px">راه‌اندازی اولیه</h2><p class="muted" style="margin:0 0 16px">نخستین حساب، مدیر آوند خواهد بود. این کار فقط از همین رایانه ممکن است.</p>${E}
<form method="post" action="/setup" id="f" autocomplete="on"><label class="field"><span>نام کاربری مدیر</span><input class="input ltr" name="username" autocomplete="username" required minlength="3" maxlength="24"></label>
<label class="field"><span>رمز عبور (حداقل ۸ نویسه)</span><input class="input ltr" type="password" name="password" autocomplete="new-password" required minlength="8"></label>${pwNote}<button class="btn pri lg" style="width:100%">ساخت حساب مدیر</button></form>`;
    if (tab === 'access') return `${tabs}${E}<form method="post" action="/register" id="f" autocomplete="on">
<label class="field"><span>کد دسترسی (از مدیر بگیرید)</span><input class="input ltr" name="code" placeholder="AVD-XXXX-XXXX" required autocapitalize="characters" autocomplete="off"></label>
<label class="field"><span>نام کاربری</span><input class="input ltr" name="username" autocomplete="username" required minlength="3" maxlength="24"></label>
<label class="field"><span>رمز عبور</span><input class="input ltr" type="password" name="password" autocomplete="new-password" required minlength="8"></label>
<label class="field"><span>تکرار رمز</span><input class="input ltr" type="password" name="password2" autocomplete="new-password" required minlength="8"></label>${pwNote}<button class="btn pri lg" style="width:100%">ساخت حساب</button></form>`;
    return `${tabs}${E}<form method="post" action="/login" id="f" autocomplete="on">
<label class="field"><span>نام کاربری</span><input class="input ltr" name="username" autocomplete="username" required></label>
<label class="field"><span>رمز عبور</span><input class="input ltr" type="password" name="password" autocomplete="current-password" required></label>${pwNote}
<button class="btn pri lg" style="width:100%">ورود</button><p class="muted" style="font-size:12.5px;margin:12px 0 0">می‌توانید از مرورگر بخواهید رمز را ذخیره کند؛ دفعه‌ی بعد خودکار پر می‌شود.</p></form>`;
  }
  function draw() {
    $('#box').innerHTML = form();
    $$('[data-t]').forEach((b) => b.addEventListener('click', () => { tab = b.dataset.t; draw(); }));
    const f = $('#f'); let ok = false;
    f.addEventListener('submit', async (e) => {
      if (ok || !st.central) return; // حالت لوکال: بدون کپچا
      e.preventDefault();
      const btn = f.querySelector('button.pri'); btn.disabled = true;
      try { await VC.ensureBrowser(); f.elements.captcha.value = await VC.solve(); ok = true; f.requestSubmit(); }
      catch { btn.disabled = false; }
    });
    VC.mountBadges();
  }
  draw();
  if (st.central) { await window.splashDone; try { VC.ensureBrowser(); } catch {} } else { $$('[data-vc-badge]').forEach((e) => e.remove()); }
})();
