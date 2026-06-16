import { useGetGoogleOauthLinkApi } from "#/api/auth/get-google-oauth-link";
import { Button } from "#/components/ui/button";

export const OauthContainer = () => {
    const { error, getGoogleOauthLink, isFetching } = useGetGoogleOauthLinkApi();

    const handleGoogleOauth = async () => {
        const result = await getGoogleOauthLink();
        const authUrl = result.data?.authUrl;
        if (authUrl) {
            window.location.assign(authUrl);
        }
    };

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
                {isFetching ? "Connecting..." : "Continue With Google"}
            </Button>
            <Button className="flex-1 items-center gap-3 flex" variant={"outline"}>
                <img src="/github.svg" alt="" className="invert size-4" /> Continue With
                Github
            </Button>
            {error ? (
                <p className="text-destructive text-xs">
                    {error.message || "Could not start Google sign in"}
                </p>
            ) : null}
        </div>
    );
};
