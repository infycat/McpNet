let monacoPromise;

// Monaco is large and is only needed when an editor is mounted. Sharing this
// promise keeps concurrent editors from requesting or initializing it twice.
export function loadMonaco() {
  if (!monacoPromise) {
    monacoPromise = import('monaco-editor').then((monaco) => {
      window.MonacoEditor = monaco;
      return monaco;
    });
  }
  return monacoPromise;
}
