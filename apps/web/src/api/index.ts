import ky, { type Options } from "ky";

type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

type ApiStatus = "success" | "fail" | "error";

type ApiErrorBody = {
	code?: string;
	message?: string;
	details?: unknown;
	[key: string]: unknown;
};

type ApiEnvelope<T> = {
	status: ApiStatus;
	data?: T;
	message?: string;
	error?: ApiErrorBody;
};

export class APIError extends Error {
	constructor(
		message: string,
		public readonly status: number = 500,
		public readonly code?: string,
		public readonly details?: unknown,
	) {
		super(message);
		this.name = "APIError";
	}
}

const writeMethods = new Set(["POST", "PUT", "PATCH", "DELETE"]);

const apiUrl = import.meta.env.VITE_API_URL ?? "http://localhost:4001";

const createIdempotencyKey = () => {
	if (globalThis.crypto?.randomUUID) {
		return globalThis.crypto.randomUUID();
	}

	return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
};

const isApiEnvelope = <T>(body: unknown): body is ApiEnvelope<T> => {
	if (!body || typeof body !== "object") {
		return false;
	}

	const status = (body as { status?: unknown }).status;
	return status === "success" || status === "fail" || status === "error";
};

const readJson = async (response: Response): Promise<unknown> => {
	if (response.status === 204) {
		return undefined;
	}

	return response
		.clone()
		.json()
		.catch(() => undefined);
};

const getError = (body: unknown, response: Response) => {
	if (isApiEnvelope(body)) {
		return {
			code: body.error?.code,
			message:
				body.error?.message ??
				body.message ??
				`HTTP ${response.status}: ${response.statusText}`,
			details: body.error?.details ?? body.error ?? body,
		};
	}

	const message =
		body && typeof body === "object" && "message" in body
			? String((body as { message?: unknown }).message)
			: `HTTP ${response.status}: ${response.statusText}`;

	return { message, details: body };
};

const baseClient = ky.create({
	prefix: apiUrl,
	credentials: "include",
	throwHttpErrors: false,
	hooks: {
		beforeRequest: [
			({ request }) => {
				if (
					writeMethods.has(request.method) &&
					!request.headers.has("Idempotency-Key")
				) {
					request.headers.set("Idempotency-Key", createIdempotencyKey());
				}
			},
		],
		afterResponse: [
			async ({ response }) => {
				if (response.ok) {
					return;
				}

				const body = await readJson(response);
				const error = getError(body, response);
				throw new APIError(
					error.message,
					response.status,
					error.code,
					error.details,
				);
			},
		],
	},
});

class APIClient {
	private client = baseClient;

	get = this.client.get.bind(this.client);
	post = this.client.post.bind(this.client);
	put = this.client.put.bind(this.client);
	patch = this.client.patch.bind(this.client);
	delete = this.client.delete.bind(this.client);

	private getMethod(method: HttpMethod) {
		const methods = {
			GET: this.client.get,
			POST: this.client.post,
			PUT: this.client.put,
			PATCH: this.client.patch,
			DELETE: this.client.delete,
		};
		return methods[method].bind(this.client);
	}

	async request<T>(
		url: string,
		method: HttpMethod = "GET",
		options?: Options,
	): Promise<T> {
		const response = await this.getMethod(method)(url, options);
		const body = await readJson(response);

		if (isApiEnvelope<T>(body)) {
			if (body.status !== "success") {
				throw new APIError(
					body.error?.message ?? body.message ?? "Request failed",
					response.status,
					body.error?.code,
					body.error?.details ?? body.error ?? body,
				);
			}

			return body.data as T;
		}

		return body as T;
	}

	async message(
		url: string,
		method: HttpMethod = "POST",
		options?: Options,
	): Promise<string | undefined> {
		const response = await this.getMethod(method)(url, options);
		const body = await readJson(response);

		if (isApiEnvelope(body)) {
			if (body.status !== "success") {
				throw new APIError(
					body.error?.message ?? body.message ?? "Request failed",
					response.status,
					body.error?.code,
					body.error?.details ?? body.error ?? body,
				);
			}

			return body.message;
		}

		return undefined;
	}
}

export const api = new APIClient();
