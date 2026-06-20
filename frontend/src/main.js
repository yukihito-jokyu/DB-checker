import "./style.css";
import "./app.css";
import { Status } from "../wailsjs/go/wails/AppHandler";

document.querySelector("#app").innerHTML = `
  <main class="app-shell">
    <section class="app-panel">
      <p class="eyebrow">DB-checker</p>
      <h1>Local database design checker</h1>
      <p class="description">
        ローカル DB にダミーデータを投入し、DB 設計の有効性を検証するための Wails アプリです。
      </p>
      <dl class="status-list" aria-label="Application status">
        <div>
          <dt>Status</dt>
          <dd id="status-ready">Checking</dd>
        </div>
        <div>
          <dt>Version</dt>
          <dd id="status-version">-</dd>
        </div>
      </dl>
    </section>
  </main>
`;

const readyElement = document.querySelector("#status-ready");
const versionElement = document.querySelector("#status-version");

Status()
  .then((status) => {
    readyElement.textContent = status.ready ? "Ready" : "Not ready";
    versionElement.textContent = status.version;
  })
  .catch(() => {
    readyElement.textContent = "Unavailable";
    versionElement.textContent = "-";
  });
