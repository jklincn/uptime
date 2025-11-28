export default {
	async fetch(request) {
		const url = new URL(request.url);

		if (url.pathname.startsWith("/api/")) {
			// 配置后端地址
			const backendOrigin = "https://uptime-backend.jklincn.com";
			const targetUrl = backendOrigin + url.pathname + url.search;

			// 复制请求头，并移除 Host 头部，让 fetch 根据 URL 自动设置
			const newHeaders = new Headers(request.headers);
			newHeaders.delete("Host");
			newHeaders.delete("Referer"); // 可选：根据需要决定是否保留 Referer

			// 创建新请求转发给后端
			const newRequest = new Request(targetUrl, {
				method: request.method,
				headers: newHeaders,
				body: request.body,
				redirect: "follow"
			});

			try {
				return await fetch(newRequest);
			} catch (e: any) {
				return new Response("Backend Error: " + e.message, { status: 502 });
			}
		}
		return new Response(null, { status: 404 });
	},
} satisfies ExportedHandler<Env>;
