(function () {
  var theme = 'light';
  try {
    var stored = localStorage.getItem('ant-theme');
    if (stored === 'light' || stored === 'dark') theme = stored;
  } catch (_) {}
  if (theme === 'dark') document.documentElement.classList.add('dark');
  else document.documentElement.classList.remove('dark');
})();
