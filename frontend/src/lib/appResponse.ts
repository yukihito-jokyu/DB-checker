import type { wails } from "@wails/go/models";

export type AppResponse<T> = {
	data?: T | null;
	error?: wails.ErrorResponse | null;
};

export class AppError extends Error {
	readonly code: string;

	/** backend の code / message を保持する frontend 用エラーを作成する。 */
	constructor(code: string, message: string) {
		super(message);
		this.name = "AppError";
		this.code = code;
	}
}

/** response が error を持つか判定する。 */
export function hasResponseError<T>(
	response: AppResponse<T>,
): response is AppResponse<T> & { error: wails.ErrorResponse } {
	return response.error != null;
}

/** backend の ErrorResponse を AppError に変換する。 */
export function toAppError(error: wails.ErrorResponse): AppError {
	return new AppError(error.code, error.message);
}

/** response から data を取り出し、欠けている場合は想定外エラーにする。 */
export function requireResponseData<T>(response: AppResponse<T>): T {
	if (response.data == null) {
		throw new AppError("UNEXPECTED", "予期しないエラーが発生しました");
	}
	return response.data;
}

/** 完全成功を期待する response から data を取り出す。 */
export function unwrapResponse<T>(response: AppResponse<T>): T {
	if (hasResponseError(response)) {
		// Wails の Promise rejection に依存せず、画面側の try/catch へ寄せる。
		throw toAppError(response.error);
	}
	return requireResponseData(response);
}
