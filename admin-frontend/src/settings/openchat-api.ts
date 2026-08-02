export interface APIResponse<T> {
	code: number;
	message?: string;
	data: T;
}

export const openchatAPIBase = (robotCode: string) =>
	`/api/v1/openchat/${encodeURIComponent(robotCode)}/robot`;

export async function openchatRequest<T>(url: string, options: RequestInit = {}): Promise<T> {
	const response = await fetch(url, {
		credentials: 'same-origin',
		headers: { 'Content-Type': 'application/json', ...options.headers },
		...options,
	});
	let payload: APIResponse<T>;
	try {
		payload = (await response.json()) as APIResponse<T>;
	} catch {
		throw new Error(`请求失败（HTTP ${response.status}）`);
	}
	if (!response.ok || payload.code !== 200) {
		throw new Error(payload.message || `请求失败（HTTP ${response.status}）`);
	}
	return payload.data;
}
