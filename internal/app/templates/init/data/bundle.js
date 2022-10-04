(() => {
  // src/index.ts
  var root = `
<main>
  <div class="section">
    <div class="container">
      <div class="status">
        <img class="logo" src="/logo.png" alt="neon logo" />
      </div>
    </div>
  </div>
</main>
`;
  server.render(root, 200);
  server.setTitle("Neon");
  server.setMeta("name", { "itemprop": "name", "content": "Neon" });
  server.setMeta("description", { "name": "description", "content": "Neon Status App" });
  server.setMeta("copyright", { "name": "copyright", "content": "Boris HUISGEN" });
  server.setMeta("generator", { "name": "generator", "content": `neon (engine='server', date='${new Date().toUTCString()}')` });
  server.setLink("canonical", { "rel": "canonical", "href": `http://localhost:8080${server.url()}` });
  server.setScript("schema-json-ld", {
    "type": "application/ld+json",
    "children": JSON.stringify({
      "@context": "http://schema.org",
      "@type": "WebSite",
      url: `http://localhost:8080${server.url()}`,
      name: "Neon",
      inLanguage: "en-us",
      description: "Neon Status App",
      keywords: "Neon",
      copyrightYear: new Date().getFullYear(),
      copyrightHolder: {
        "@type": "Person",
        name: "Boris HUISGEN"
      }
    })
  });
})();
