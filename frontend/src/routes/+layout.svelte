<script>
  import '../app.css';
</script>

<div class="app">
  <header>
    <h1>🚀 API Tester</h1>
    <p>A tool for testing HTTP APIs</p>
  </header>
  
  <main>
    <slot />
  </main>
</div>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background-color: #f5f5f5;
  }

  .app {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  header {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 2rem;
    text-align: center;
    box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  }

  header h1 {
    margin: 0;
    font-size: 2.125rem;
    font-weight: 600;
  }

  header p {
    margin: 0.5rem 0 0 0;
    opacity: 0.9;
    font-size: 0.95rem;
  }

  main {
    flex: 1;
    padding: 2rem 0;
    width: 100%;
    box-sizing: border-box;
  }
</style>