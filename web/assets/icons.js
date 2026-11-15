// آیکون‌های داخلی (بدون وابستگی به اینترنت)
(function () {
  const P = {
    upload: 'M12 16V4m0 0L8 8m4-4 4 4M4 16v3a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-3',
    download: 'M12 4v12m0 0-4-4m4 4 4-4M4 20h16',
    image: 'M4 5h16v14H4zM8.5 10a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3M4 17l5-5 4 4 3-3 4 4',
    video: 'M4 6h11v12H4zM15 10l5-3v10l-5-3',
    doc: 'M7 3h7l5 5v13H7zM14 3v5h5M10 13h6M10 17h6',
    file: 'M7 3h7l5 5v13H7zM14 3v5h5',
    folder: 'M3 6h6l2 2h10v11H3z',
    folderPlus: 'M3 6h6l2 2h10v11H3zM12 11v5m-2.5-2.5h5',
    search: 'M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14M20 20l-4-4',
    settings: 'M4 7h10M18 7h2M4 17h2M10 17h10M14 5v4M6 15v4',
    user: 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8M4 20c0-4 4-6 8-6s8 2 8 6',
    users: 'M9 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7M2.5 20c0-3.5 3-5.5 6.5-5.5s6.5 2 6.5 5.5M16 4.5a3.5 3.5 0 0 1 0 6.5M18 14.5c2 .6 3.5 2.2 3.5 5.5',
    logout: 'M9 4H5v16h4M16 8l4 4-4 4M20 12H9',
    trash: 'M5 7h14M10 7V4h4v3M7 7l1 13h8l1-13',
    plus: 'M12 5v14M5 12h14',
    x: 'M6 6l12 12M18 6 6 18',
    check: 'M5 12l5 5 9-10',
    chevL: 'M15 5l-7 7 7 7',
    chevR: 'M9 5l7 7-7 7',
    grid: 'M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM13 13h7v7h-7z',
    list: 'M8 6h12M8 12h12M8 18h12M4 6h.01M4 12h.01M4 18h.01',
    play: 'M8 5l11 7-11 7z',
    music: 'M9 18V5l11-2v13M9 18a3 3 0 1 1-6 0 3 3 0 0 1 6 0M20 16a3 3 0 1 1-6 0 3 3 0 0 1 6 0',
    heart: 'M12 20s-8-5-8-11a4.5 4.5 0 0 1 8-2.5A4.5 4.5 0 0 1 20 9c0 6-8 11-8 11z',
    sun: 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8M12 2v2M12 20v2M2 12h2M20 12h2M5 5l1.5 1.5M17.5 17.5 19 19M5 19l1.5-1.5M17.5 6.5 19 5',
    moon: 'M20 14A8 8 0 1 1 10 4a7 7 0 0 0 10 10z',
    help: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18M9.5 9.5a2.5 2.5 0 1 1 4 2c-1 .7-1.5 1-1.5 2M12 17h.01',
    drive: 'M4 14h16v6H4zM4 14l3-9h10l3 9M8 17h.01',
    key: 'M15 9a4 4 0 1 0-3.9 5L4 21h3v-2h2v-2h2l1.2-1.2A4 4 0 0 0 15 9zM16 7h.01',
    link: 'M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1',
    more: 'M5 12h.01M12 12h.01M19 12h.01',
    edit: 'M4 20h4L19 9l-4-4L4 16z',
    shield: 'M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6zM9 12l2 2 4-4',
    refresh: 'M20 11a8 8 0 1 0-2 6M20 4v7h-7',
    arrowR: 'M5 12h14M13 6l6 6-6 6',
    copy: 'M8 8h11v12H8zM5 16V4h11',
    home: 'M4 11l8-7 8 7v9H4z',
    wifi: 'M2 9a15 15 0 0 1 20 0M5 12.5a10 10 0 0 1 14 0M8.5 16a5 5 0 0 1 7 0M12 19.5h.01',
    info: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18M12 11v5M12 8h.01',
    globe: 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18',
    menu: 'M4 7h16M4 12h16M4 17h16',
    move: 'M3 6h6l2 2h10v11H3zM10 14h6m-2.5-2.5L16 14l-2.5 2.5',
    sort: 'M7 4v16m0 0-3-3m3 3 3-3M17 20V4m0 0-3 3m3-3 3 3',
  };
  window.ic = (n, cls = '') => `<svg class="ic ${cls}" viewBox="0 0 24 24" aria-hidden="true"><path d="${P[n] || P.file}"/></svg>`;
  // لوگوی آوند: ظرف/قطره با نوارهای رگ‌مانند
  window.avandLogo = (s = 44) => `<svg class="logo" width="${s}" height="${s}" viewBox="0 0 64 64" aria-label="آوند">
<defs><linearGradient id="avg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#2dd4bf"/><stop offset="1" stop-color="#38bdf8"/></linearGradient>
<clipPath id="avc"><path d="M32 6C32 6 14 26 14 40a18 18 0 0 0 36 0C50 26 32 6 32 6z"/></clipPath></defs>
<rect x="2" y="2" width="60" height="60" rx="18" fill="#0b1b1d" stroke="url(#avg)" stroke-opacity=".5"/>
<g transform="translate(5 4) scale(.84)"><path d="M32 6C32 6 14 26 14 40a18 18 0 0 0 36 0C50 26 32 6 32 6z" fill="url(#avg)"/>
<g clip-path="url(#avc)" fill="none" stroke="#0b1b1d" stroke-opacity=".55" stroke-width="2.6"><path d="M10 34h44M10 43h44M10 52h44"/></g>
<circle cx="32" cy="22" r="3.2" fill="#f5b83d"/></g></svg>`;
})();
