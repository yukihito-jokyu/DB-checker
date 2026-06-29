import { useCallback, useEffect, useState } from "react";

import { type AppConfig, getAppConfig } from "@/app/services/config";
import { type AppStatus, getAppStatus } from "@/app/services/status";
import { Button } from "@/components/ui/button";

type LoadState = "idle" | "loading" | "success" | "error";

export function HomePage() {
	const [status, setStatus] = useState<AppStatus | null>(null);
	const [config, setConfig] = useState<AppConfig | null>(null);
	const [loadState, setLoadState] = useState<LoadState>("idle");

	const loadAppData = useCallback(async () => {
		// 疎通確認の再試行時も同じ状態遷移で扱う。
		setLoadState("loading");

		try {
			const [nextStatus, nextConfig] = await Promise.all([
				getAppStatus(),
				getAppConfig(),
			]);
			setStatus(nextStatus);
			setConfig(nextConfig);
			setLoadState("success");
		} catch {
			setStatus(null);
			setConfig(null);
			setLoadState("error");
		}
	}, []);

	useEffect(() => {
		void loadAppData();
	}, [loadAppData]);

	const readyText =
		loadState === "error"
			? "Unavailable"
			: status?.ready
				? "Ready"
				: "Not ready";
	const versionText = loadState === "error" ? "-" : (status?.version ?? "-");
	const configVersionText =
		loadState === "error" ? "-" : (config?.version.toString() ?? "-");
	const profileCountText =
		loadState === "error" ? "-" : (config?.connectionProfiles.length ?? "-");

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
					<div className="rounded-lg border border-[#d9e0ea] p-3.5">
						<dt className="text-xs font-bold uppercase text-[#526171]">
							Config
						</dt>
						<dd className="mt-1.5 text-base font-bold text-[#17202a]">
							{loadState === "loading" ? "Loading" : configVersionText}
						</dd>
					</div>
					<div className="rounded-lg border border-[#d9e0ea] p-3.5">
						<dt className="text-xs font-bold uppercase text-[#526171]">
							Profiles
						</dt>
						<dd className="mt-1.5 text-base font-bold text-[#17202a]">
							{loadState === "loading" ? "-" : profileCountText}
						</dd>
					</div>
				</dl>
				<div className="mt-6">
					<Button
						type="button"
						onClick={loadAppData}
						disabled={loadState === "loading"}
					>
						{loadState === "loading" ? "確認中" : "再試行"}
					</Button>
				</div>
			</section>
		</main>
	);
}
