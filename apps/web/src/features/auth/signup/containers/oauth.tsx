import { useGetGithubOauthLinkApi } from "#/api/auth/get-github-oauth-link";
import { useGetGoogleOauthLinkApi } from "#/api/auth/get-google-oauth-link";
import { Button } from "#/components/ui/button";

export const OauthContainer = () => {
	const {
		error: googleError,
		getGoogleOauthLink,
		isFetching: isGoogleFetching,
	} = useGetGoogleOauthLinkApi();
	const {
		error: githubError,
		getGithubOauthLink,
		isFetching: isGithubFetching,
	} = useGetGithubOauthLinkApi();

	const handleGoogleOauth = async () => {
		const result = await getGoogleOauthLink();
		const authUrl = result.data?.authUrl;
		if (authUrl) {
			window.location.assign(authUrl);
		}
	};

	const handleGithubOauth = async () => {
		const result = await getGithubOauthLink();
		const authUrl = result.data?.authUrl;
		if (authUrl) {
			window.location.assign(authUrl);
		}
	};

	const error = googleError ?? githubError;
	const isFetching = isGoogleFetching || isGithubFetching;

	return (
		<div className="flex w-full gap-4 flex-col">
			<Button
				className="flex-1 items-center gap-3 flex"
				disabled={isFetching}
				onClick={handleGoogleOauth}
				type="button"
				variant={"outline"}
			>
				<img src="/google.svg" alt="" className="invert size-4" />
				{isGoogleFetching ? "Connecting..." : "Continue With Google"}
			</Button>
			<Button
				className="flex-1 items-center gap-3 flex"
				disabled={isFetching}
				onClick={handleGithubOauth}
				type="button"
				variant={"outline"}
			>
				<img src="/github.svg" alt="" className="invert size-4" />
				{isGithubFetching ? "Connecting..." : "Continue With Github"}
			</Button>
			{error ? (
				<p className="text-destructive text-xs">
					{error.message || "Could not start OAuth sign in"}
				</p>
			) : null}
		</div>
	);
};
