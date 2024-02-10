package neon

import (
	_ "github.com/bhuisgen/neon/pkg/modules/app/store/storage/memory"

	_ "github.com/bhuisgen/neon/pkg/modules/app/fetcher/providers/file"
	_ "github.com/bhuisgen/neon/pkg/modules/app/fetcher/providers/rest"

	_ "github.com/bhuisgen/neon/pkg/modules/app/loader/parsers/json"
	_ "github.com/bhuisgen/neon/pkg/modules/app/loader/parsers/raw"

	_ "github.com/bhuisgen/neon/pkg/modules/app/server/listener/listeners/local"
	_ "github.com/bhuisgen/neon/pkg/modules/app/server/listener/listeners/redirect"
	_ "github.com/bhuisgen/neon/pkg/modules/app/server/listener/listeners/tls"

	_ "github.com/bhuisgen/neon/pkg/modules/app/server/site/middlewares/compress"
	_ "github.com/bhuisgen/neon/pkg/modules/app/server/site/middlewares/header"
	_ "github.com/bhuisgen/neon/pkg/modules/app/server/site/middlewares/logger"
	_ "github.com/bhuisgen/neon/pkg/modules/app/server/site/middlewares/rewrite"
	_ "github.com/bhuisgen/neon/pkg/modules/app/server/site/middlewares/static"

	_ "github.com/bhuisgen/neon/pkg/modules/app/server/site/handlers/file"
	_ "github.com/bhuisgen/neon/pkg/modules/app/server/site/handlers/js"
	_ "github.com/bhuisgen/neon/pkg/modules/app/server/site/handlers/robots"
	_ "github.com/bhuisgen/neon/pkg/modules/app/server/site/handlers/sitemap"
)
