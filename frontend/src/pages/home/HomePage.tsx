import { useCallback, useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Status } from "../../../wailsjs/go/wails/AppHandler";

type AppStatus = {
	ready: boolean;
	version: string;
};

type LoadState = "idle" | "loading" | "success" | "error";

export function HomePage() {
	const [status, setStatus] = useState<AppStatus | null>(null);
	const [loadState, setLoadState] = useState<LoadState>("idle");

	const loadStatus = useCallback(async () => {
		setLoadState("loading");

		try {
			const nextStatus = await Status();
			setStatus(nextStatus);
			setLoadState("success");
		} catch {
			setStatus(null);
			setLoadState("error");
		}
	}, []);

	useEffect(() => {
		void loadStatus();
	}, [loadStatus]);

	const readyText =
		loadState === "error"
			? "Unavailable"
			: status?.ready
				? "Ready"
				: "Not ready";
	const versionText = loadState === "error" ? "-" : (status?.version ?? "-");

	return (
		<main className="grid min-h-screen place-items-center bg-[#f5f7fb] p-8 text-[#17202a]">
			<section className="w-full max-w-[680px] rounded-lg border border-[#d9e0ea] bg-white p-8 shadow-[0_18px_45px_rgba(23,32,42,0.08)]">
				<p className="mb-2.5 text-[13px] font-bold uppercase text-[#3d6f9f]">
					DB-checker
				</p>
				<h1 className="m-0 text-[28px] font-bold leading-tight">
					Local database design checker
				</h1>
				<p className="mt-4 text-[15px] leading-7 text-[#526171]">
					ローカル DB にダミーデータを投入し、DB 設計の有効性を検証するための
					Wails アプリです。
				</p>
				<dl
					className="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-2"
					aria-label="Application status"
				>
					<div className="rounded-lg border border-[#d9e0ea] p-3.5">
						<dt className="text-xs font-bold uppercase text-[#526171]">
							Status
						</dt>
						<dd className="mt-1.5 text-base font-bold text-[#17202a]">
							{loadState === "loading" ? "Checking" : readyText}
						</dd>
					</div>
					<div className="rounded-lg border border-[#d9e0ea] p-3.5">
						<dt className="text-xs font-bold uppercase text-[#526171]">
							Version
						</dt>
						<dd className="mt-1.5 text-base font-bold text-[#17202a]">
							{versionText}
						</dd>
					</div>
				</dl>
				<div className="mt-6">
					<Button
						type="button"
						onClick={loadStatus}
						disabled={loadState === "loading"}
					>
						{loadState === "loading" ? "確認中" : "再試行"}
					</Button>
				</div>
			</section>
		</main>
	);
}
