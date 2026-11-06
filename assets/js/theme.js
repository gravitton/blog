(function () {
  const storageKey = "theme";
  const system = window.matchMedia("(prefers-color-scheme: dark)");

  function systemTheme() {
    return system.matches ? "dark" : "light";
  }

  function storedTheme() {
    try {
      return localStorage.getItem(storageKey);
    } catch {
      return null;
    }
  }

  function store(theme) {
    try {
      if (theme === systemTheme()) {
        localStorage.removeItem(storageKey);
      } else {
        localStorage.setItem(storageKey, theme);
      }
    } catch {
      return;
    }
  }

  function currentTheme() {
    return storedTheme() ?? systemTheme();
  }

  function apply() {
    const theme = currentTheme();
    document.documentElement.dataset.theme = theme;
    for (const element of document.querySelectorAll("[data-scheme]")) {
      element.media = element.dataset.scheme === theme ? "all" : "not all";
    }
  }

  function toggle() {
    store(currentTheme() === "dark" ? "light" : "dark");
    apply();
  }

  function bind() {
    apply();
    const button = document.querySelector(".theme-toggle");
    button.hidden = false;
    button.addEventListener("click", toggle);
  }

  apply();
  system.addEventListener("change", apply);
  document.addEventListener("DOMContentLoaded", bind);
})();
