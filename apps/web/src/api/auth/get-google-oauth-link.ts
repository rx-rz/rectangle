import { useQuery } from "@tanstack/react-query";
import { api } from "..";
import { API_ROUTES } from "../api-routes";
import { queries } from "../queries";

export type GetGoogleOauthLinkOutput = {
	authUrl: string;
};

export const getGoogleOauthLinkApi = async () => {
	return await api.request<GetGoogleOauthLinkOutput>(
		API_ROUTES.auth.oauth.google,
	);
};

export const useGetGoogleOauthLinkApi = () => {
	const query = useQuery({
		queryKey: queries.auth.getGoogleOauthLink.queryKey,
		queryFn: getGoogleOauthLinkApi,
		enabled: false,
	});

	const { data, error, isFetching, refetch } = query;
	return {
		...query,
		data,
		error,
		isFetching,
		getGoogleOauthLink: refetch,
	};
};
