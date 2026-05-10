import './App.scss'

function App() {
  return (
    <div className="app-layout">
      <aside className="app-sidebar" aria-label="Primary navigation">
        <div className="app-sidebar-header">
          <span className="app-sidebar-title">Mod Manager</span>
        </div>

        <nav className="app-sidebar-navigation">
          <a className="app-sidebar-link app-sidebar-link-active" href="#mods">
            Mods
          </a>
          <a className="app-sidebar-link" href="#profiles">
            Profiles
          </a>
          <a className="app-sidebar-link" href="#settings">
            Settings
          </a>
        </nav>
      </aside>

      <main className="app-main">
        <div className="app-main-header">
          <h1 className="app-main-title">Mods</h1>
          <p className="app-main-description">
            Select a game profile to review installed mods.
          </p>
        </div>
      </main>
    </div>
  )
}

export default App
