import { useEffect, useState } from "react";
import DOMPurify from "dompurify";

const API_BASE = "https://backend.terimasurel.dpdns.org";
const PAGE_SIZE = 5;
const DB_NAME = "MailConsoleDB";
const STORE_NAME = "emails";

// Cache the IndexedDB connection to avoid reinitializing
let dbConnection = null;

// Decode HTML entities like &lt; &gt; &quot; etc
const decodeHTML = (html) => {
  const textarea = document.createElement("textarea");
  textarea.innerHTML = html;
  return textarea.value;
};

// IndexedDB helper functions
const initDB = () => {
  if (dbConnection) {
    return Promise.resolve(dbConnection);
  }

  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1);

    request.onerror = () => {
      console.log("Error opening IndexedDB", request.error);
      reject(request.error);
    };

    request.onsuccess = () => {
      dbConnection = request.result;
      resolve(dbConnection);
    };

    request.onupgradeneeded = (event) => {
      const db = event.target.result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME, { keyPath: "id" });
        console.log("IndexedDB object store created");
      }
    };
  });
};

const getCachedEmail = async (id) => {
  try {
    const db = await initDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction(STORE_NAME, "readonly");
      const store = transaction.objectStore(STORE_NAME);
      const request = store.get(id);

      request.onerror = () => {
        console.log("Error reading from IndexedDB", request.error);
        reject(request.error);
      };

      request.onsuccess = () => {
        resolve(request.result);
      };
    });
  } catch (err) {
    console.log("Failed to get cached email", err);
    return null;
  }
};

const cacheEmail = async (email) => {
  try {
    const db = await initDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction(STORE_NAME, "readwrite");
      const store = transaction.objectStore(STORE_NAME);
      const request = store.put(email);

      request.onerror = () => {
        console.log("Error writing to IndexedDB", request.error);
        reject(request.error);
      };

      request.onsuccess = () => {
        console.log(`Cached email ${email.id}`);
        resolve(true);
      };
    });
  } catch (err) {
    console.log("Failed to cache email", err);
    return null;
  }
};

const clearAllCache = async () => {
  try {
    const db = await initDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction(STORE_NAME, "readwrite");
      const store = transaction.objectStore(STORE_NAME);
      const request = store.clear();

      request.onerror = () => {
        console.log("Error clearing IndexedDB", request.error);
        reject(request.error);
      };

      request.onsuccess = () => {
        console.log("IndexedDB cache cleared");
        resolve(true);
      };
    });
  } catch (err) {
    console.log("Failed to clear cache", err);
    return null;
  }
};

const getDBSize = async () => {
  try {
    const db = await initDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction(STORE_NAME, "readonly");
      const store = transaction.objectStore(STORE_NAME);
      const request = store.count();

      request.onerror = () => {
        console.log("Error counting IndexedDB items", request.error);
        reject(request.error);
      };

      request.onsuccess = () => {
        resolve(request.result);
      };
    });
  } catch (err) {
    console.log("Failed to get cache size", err);
    return 0;
  }
};

const createIframeHTML = (html, isDarkMode = false) => {
  const decoded = decodeHTML(html);
  const sanitized = DOMPurify.sanitize(decoded);
  const bgColor = isDarkMode ? "#0f1a2e" : "#ffffff";
  const textColor = isDarkMode ? "#e8edf7" : "#333";
  
  return `
    <!DOCTYPE html>
    <html>
    <head>
      <meta charset="UTF-8">
      <meta name="viewport" content="width=device-width, initial-scale=1.0">
      <style>
        * {
          max-width: 100% !important;
          box-sizing: border-box !important;
        }
        html, body {
          width: 100% !important;
          height: 100% !important;
          margin: 0 !important;
          padding: 0 !important;
        }
        body {
          font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen-Sans, Ubuntu, Cantarell, "Helvetica Neue", sans-serif !important;
          font-size: 14px !important;
          line-height: 1.6 !important;
          color: ${textColor} !important;
          background: ${bgColor} !important;
          overflow-y: auto !important;
          overflow-x: hidden !important;
          padding: 12px !important;
          word-wrap: break-word !important;
          overflow-wrap: break-word !important;
        }
        img {
          max-width: 100% !important;
          width: 100% !important;
          height: auto !important;
          display: block !important;
          margin: 8px 0 !important;
        }
        table {
          width: 100% !important;
          max-width: 100% !important;
        }
        td, th {
          max-width: 100% !important;
          word-wrap: break-word !important;
        }
        a {
          color: #0066cc !important;
        }
        pre {
          background: ${isDarkMode ? "#0b1324" : "#f5f5f5"} !important;
          padding: 10px !important;
          border-radius: 4px !important;
          overflow-x: auto !important;
          color: ${isDarkMode ? "#e8edf7" : "#333"} !important;
          max-width: 100% !important;
          width: 100% !important;
        }
        code {
          background: ${isDarkMode ? "#0b1324" : "#f5f5f5"} !important;
          padding: 2px 6px !important;
          border-radius: 3px !important;
          font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace !important;
          color: ${isDarkMode ? "#e8edf7" : "#333"} !important;
        }
        blockquote {
          border-left: 3px solid ${isDarkMode ? "#22314f" : "#ddd"} !important;
          margin: 0 !important;
          padding-left: 12px !important;
          color: ${isDarkMode ? "#9fb1cc" : "#666"} !important;
        }
      </style>
    </head>
    <body>
      ${sanitized}
    </body>
    </html>
  `;
};

export default function App() {
  // Try to restore logged-in state from localStorage
  const [activeMenu, setActiveMenu] = useState("home");
  const [messages, setMessages] = useState([]);
  const [selectedMessageId, setSelectedMessageId] = useState("");
  const [selectedMessageDetail, setSelectedMessageDetail] = useState(null);
  const [emailAddress, setEmailAddress] = useState("");
  const [openedEmail, setOpenedEmail] = useState(() => {
    try {
      return localStorage.getItem("openedEmail") || "";
    } catch (err) {
      console.log("Failed to restore openedEmail from localStorage", err);
      return "";
    }
  });
  const [loading, setLoading] = useState(false);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [error, setError] = useState("");
  const [validationError, setValidationError] = useState(null);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const [theme, setTheme] = useState("dark");
  const [detailSource, setDetailSource] = useState("");

  const handleOpenInbox = async () => {
    const trimmed = emailAddress.trim();
    if (!trimmed) return;

    setValidationError(null);
    setLoading(true);

    try {
      // Validate domain first
      const validateUrl = `${API_BASE}/domain/validate?email=${encodeURIComponent(trimmed)}`;
      const validateResponse = await fetch(validateUrl);
      const validationResult = await validateResponse.json();

      if (validationResult.status === "error") {
        setValidationError(validationResult);
        setLoading(false);
        return;
      }

      // If validation passed, proceed to open inbox
      setOpenedEmail(trimmed);
      try {
        localStorage.setItem("openedEmail", trimmed);
      } catch (err) {
        console.log("Failed to save openedEmail to localStorage", err);
      }
      setPage(1);
      void fetchInbox(trimmed, 1);
    } catch (err) {
      console.log("Validation error", err);
      setValidationError({
        status: "error",
        message: err?.message ?? "Failed to validate domain",
      });
      setLoading(false);
    }
  };

  const fetchInbox = async (address, page) => {
    setLoading(true);
    setError("");
    try {
      const url = `${API_BASE}/inbox?email=${encodeURIComponent(address)}&page=${page}`;
      const response = await fetch(url);

      if (!response.ok) {
        throw new Error("Failed to load inbox.");
      }

      const data = await response.json();
      const incoming = Array.isArray(data) ? data : data?.messages ?? [];
      setMessages(incoming);
      setHasMore(incoming.length >= PAGE_SIZE);
      setSelectedMessageId("");
      setSelectedMessageDetail(null);
    } catch (err) {
      setError(err?.message ?? "Failed to load inbox.");
      setMessages([]);
      setSelectedMessageId("");
      setSelectedMessageDetail(null);
    } finally {
      setLoading(false);
    }
  };

  const fetchEmailDetail = async (id) => {
    setLoadingDetail(true);
    try {
      let cached = null;
      try {
        cached = await getCachedEmail(id);
      } catch (err) {
        console.log("IndexedDB read failed, will fetch from API", err);
      }

      if (cached) {
        setSelectedMessageDetail(cached);
        setDetailSource("cache");
        setLoadingDetail(false);
        return;
      }

      const url = `${API_BASE}/email?id=${encodeURIComponent(id)}`;
      const response = await fetch(url);

      if (!response.ok) {
        throw new Error("Failed to load email.");
      }

      const data = await response.json();
      try {
        await cacheEmail(data);
      } catch (err) {
        console.log("IndexedDB write failed, still showing email", err);
      }
      setSelectedMessageDetail(data);
      setDetailSource("fresh");
    } catch (err) {
      console.log("Failed to load email detail", err);
      setSelectedMessageDetail(null);
      setDetailSource("");
    } finally {
      setLoadingDetail(false);
    }
  };

  const handleSelectMessage = (id) => {
    setSelectedMessageId(id);
    setDetailSource("");
    void fetchEmailDetail(id);
  };

  const handleRefreshInbox = () => {
    if (!openedEmail) return;
    void fetchInbox(openedEmail, page);
  };

  const handlePrev = () => {
    if (page <= 1) return;
    const newPage = page - 1;
    setPage(newPage);
    void fetchInbox(openedEmail, newPage);
  };

  const handleNext = () => {
    if (!hasMore) return;
    const newPage = page + 1;
    setPage(newPage);
    void fetchInbox(openedEmail, newPage);
  };

  const handleLogout = () => {
    setOpenedEmail("");
    try {
      localStorage.removeItem("openedEmail");
    } catch (err) {
      console.log("Failed to clear openedEmail from localStorage", err);
    }
    setMessages([]);
    setSelectedMessageId("");
    setSelectedMessageDetail(null);
    setError("");
    setPage(1);
    setHasMore(true);
    setDetailSource("");
  };

  // Restore inbox data only when the inbox view is active
  useEffect(() => {
    if (activeMenu === "inbox" && openedEmail) {
      void fetchInbox(openedEmail, 1);
    }
  }, [activeMenu, openedEmail]);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  const handleToggleTheme = () => {
    setTheme((prev) => (prev === "light" ? "dark" : "light"));
  };

  const handleClearCache = async () => {
    try {
      const count = await getDBSize();
      await clearAllCache();
      setSelectedMessageDetail(null);
      setDetailSource("");
      console.log(`Cleared ${count} cached emails from IndexedDB`);
    } catch (err) {
      console.log("Failed to clear cache", err);
    }
  };

  return (
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-icon">📨</div>
          <div>
            <p className="brand-title">Mail Console</p>
            <p className="brand-subtitle">Read-only</p>
          </div>
        </div>

        <nav className="menu">
          <button
            type="button"
            className={`menu-item ${activeMenu === "home" ? "active" : ""}`}
            onClick={() => setActiveMenu("home")}
          >
            Home
          </button>
          <button
            type="button"
            className={`menu-item ${activeMenu === "inbox" ? "active" : ""}`}
            onClick={() => setActiveMenu("inbox")}
          >
            Inbox
          </button>
        </nav>

        <div className="sidebar-card">
          <p className="sidebar-card-title">System Status</p>
          <div className="status">
            <span className="status-dot" />
            Operating normally
          </div>
          <p className="sidebar-card-note">
            Last sync 2 minutes ago
          </p>
        </div>
      </aside>

      <main className="content">
        <header className="header">
          <div>
            <h1>
              {activeMenu === "home"
                ? "Read-only email server"
                : activeMenu === "inbox"
                ? "Inbox"
                : "Config"}
            </h1>
            <p className="subheading">
              {activeMenu === "home"
                ? "Create a disposable mailbox instantly. No sign-up, no password."
                : openedEmail
                ? (
                    <>
                      View messages received for <strong>{openedEmail}</strong>
                    </>
                  )
                : ""}
            </p>
          </div>
          <div className="header-actions">
            <div className="pill">Read-only mode</div>
            <button className="theme-toggle" type="button" onClick={handleToggleTheme}>
              {theme === "light" ? "🌙" : "☀️"}
            </button>
            <button className="theme-toggle" type="button" onClick={handleClearCache}>
              Clear Cache
            </button>
            {openedEmail && activeMenu === "inbox" && (
              <button className="secondary" type="button" onClick={handleLogout}>
                Logout
              </button>
            )}
            <button className="ghost" type="button" disabled>
              Compose
            </button>
          </div>
        </header>

        {activeMenu === "home" ? (
          <section className="home">
            <div className="home-quickstart">
              <div>
                <p className="home-eyebrow">Quick start</p>
                <h3>Use our free domain</h3>
                <p className="home-text">
                  <strong>test@terimasurel.dpdns.org</strong> works immediately.
                  Open Inbox and enter that address to view mail.
                </p>
              </div>
              <button
                type="button"
                className="secondary"
                onClick={() => setActiveMenu("inbox")}
              >
                Open Inbox
              </button>
            </div>
            <div className="home-infographic">
              <div className="infographic-step">
                <div className="step-icon">🌍</div>
                <div>
                  <p className="step-title">Global email flow</p>
                  <p className="step-text">
                    Any sender in the world can email your domain.
                  </p>
                </div>
              </div>
              <div className="infographic-step">
                <div className="step-icon">🧭</div>
                <div>
                  <p className="step-title">DNS points to us</p>
                  <p className="step-text">
                    Set MX and A records so mail routes to this server.
                  </p>
                </div>
              </div>
              <div className="infographic-step">
                <div className="step-icon">📥</div>
                <div>
                  <p className="step-title">We receive mail</p>
                  <p className="step-text">
                    Incoming messages are captured for safe viewing.
                  </p>
                </div>
              </div>
              <div className="infographic-step">
                <div className="step-icon">👀</div>
                <div>
                  <p className="step-title">Read-only Inbox</p>
                  <p className="step-text">
                    Open the Inbox to review messages without edits.
                  </p>
                </div>
              </div>
            </div>
            <div className="home-bottom">
              <div className="home-instructions">
                <h3>Important notes</h3>
                <ul>
                  <li>
                    We only receive emails up to <strong>512 KB</strong>.
                  </li>
                  <li>
                    Attachments are not supported yet, but planned for a future update.
                  </li>
                  <li>
                    We do not support any activity that violates local or international law.
                  </li>
                  <li>
                    Emails sent to this platform are public, which means anyone can read them.
                  </li>
                  <li>
                    For more features or a custom setup: <a href="mailto:contact@kamiladigital.com">contact@kamiladigital.com</a>
                  </li>
                </ul>
              </div>
              <div className="home-panel">
                <div>
                  <p className="panel-label">How it works</p>
                  <p className="panel-title">Point your DNS to this server</p>
                </div>
                <table className="dns-table">
                  <tbody>
                    <tr>
                      <td className="dns-type">MX</td>
                      <td>&lt;yourdomain.com&gt;</td>
                      <td>mx1.terimasurel.dpdns.org</td>
                    </tr>
                  </tbody>
                </table>
                <p className="dns-note">Example of this server</p>
                <table className="dns-table">
                  <tbody>
                    <tr>
                      <td className="dns-type">MX</td>
                      <td>terimasurel.dpdns.org</td>
                      <td>mx1.terimasurel.dpdns.org</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>
        ) : activeMenu === "inbox" ? (
          <section className="inbox">
            {!openedEmail ? (
              <div className="inbox-gate">
                <h3>Open a mailbox</h3>
                <p className="config-note">
                  Enter the email address to load messages captured for that
                  mailbox.
                </p>
                {validationError && (
                  <div className="validation-error">
                    <p className="error-message">{validationError.message}</p>
                    {validationError.a_record && (
                      <p className="error-detail">
                        <strong>A Record:</strong> {validationError.a_record}
                      </p>
                    )}
                    {validationError.mx_record && (
                      <p className="error-detail">
                        <strong>MX Record:</strong> {validationError.mx_record}
                      </p>
                    )}
                  </div>
                )}
                <label className="config-label" htmlFor="inboxEmail">
                  Email address
                </label>
                <input
                  id="inboxEmail"
                  className="config-input"
                  type="email"
                  value={emailAddress}
                  onChange={(event) => setEmailAddress(event.target.value)}
                />
                <p className="config-helper">Example: test@terimasurel.dpdns.org</p>
                <button 
                  className="primary" 
                  type="button" 
                  onClick={handleOpenInbox}
                  disabled={loading}
                >
                  {loading ? "Validating..." : "Open inbox"}
                </button>
              </div>
            ) : (
              <>
                <div className="message-list">
                  <div className="list-toolbar">
                    <button
                      type="button"
                      className="refresh-button"
                      onClick={handleRefreshInbox}
                      disabled={loading}
                    >
                      <span className={`refresh-icon ${loading ? "spinning" : ""}`}>
                        ⟳
                      </span>
                      {loading ? "Refreshing..." : "Refresh"}
                    </button>
                    <div className="pagination">
                      <button
                        type="button"
                        className="page-button"
                        onClick={handlePrev}
                        disabled={page <= 1 || loading}
                      >
                        ← Prev
                      </button>
                      <span className="page-info">Page {page}</span>
                      <button
                        type="button"
                        className="page-button"
                        onClick={handleNext}
                        disabled={!hasMore || loading}
                      >
                        Next →
                      </button>
                    </div>
                  </div>
                  {error ? (
                    <div className="empty-mailbox">
                      <div className="empty-icon">⚠️</div>
                      <p>{error}</p>
                    </div>
                  ) : loading ? (
                    <div className="empty-mailbox">
                      <div className="empty-icon spinning">⟳</div>
                      <p>Loading...</p>
                    </div>
                  ) : messages.length === 0 ? (
                    <div className="empty-mailbox">
                      <div className="empty-icon">📭</div>
                      <p>No emails yet...</p>
                    </div>
                  ) : (
                    messages.map((message) => (
                      <button
                        type="button"
                        key={message.id}
                        className={`message-item ${
                          selectedMessageId === message.id ? "selected" : ""
                        }`}
                        onClick={() => handleSelectMessage(message.id)}
                      >
                        <div className="message-header">
                          <p className="message-subject">{message.subject ?? "(No subject)"}</p>
                          <span className="message-date">{message.date ?? ""}</span>
                        </div>
                        <p className="message-from">{message.from ?? "Unknown sender"}</p>
                      </button>
                    ))
                  )}
                </div>

                <div className="message-preview-panel">
                  {loadingDetail ? (
                    <div className="empty-mailbox">
                      <div className="empty-icon spinning">⟳</div>
                      <p>Loading email...</p>
                    </div>
                  ) : selectedMessageDetail ? (
                    <>
                      <div className="preview-header">
                        <div>
                          <p className="preview-subject">
                            {selectedMessageDetail.subject ?? "(No subject)"}
                          </p>
                          <p className="preview-from">{selectedMessageDetail.from}</p>
                        </div>
                        <div className="preview-meta">
                          <span>{selectedMessageDetail.date}</span>
                          {detailSource && (
                            <span className={`detail-badge ${detailSource}`}>
                              {detailSource === "cache" ? "Cached" : "Fresh"}
                            </span>
                          )}
                        </div>
                      </div>
                      {selectedMessageDetail.body ? (
                        <iframe
                          className="preview-iframe"
                          srcDoc={createIframeHTML(
                            selectedMessageDetail.body,
                            theme === "dark"
                          )}
                          title="Email content"
                          sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-modals allow-presentation"
                          style={{ border: "none", width: "100%", height: "100%" }}
                        />
                      ) : (
                        <p className="empty">No email content available.</p>
                      )}
                    </>
                  ) : (
                    <p className="empty">Select a message to preview.</p>
                  )}
                </div>
              </>
            )}
          </section>
        ) : null}
      </main>
    </div>
  );
}
