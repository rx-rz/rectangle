import { Button } from "#/components/ui/button"

export const OauthContainer = () => {
    return <div className="flex w-full gap-4 flex-col">
        <Button className="flex-1 items-center gap-3 flex" variant={"outline"}>
            <img src="/google.svg" alt="" className="invert size-4" />Continue With Google
        </Button>
        <Button className="flex-1 items-center gap-3 flex" variant={"outline"}>
            <img src="/github.svg" alt="" className="invert size-4" /> Continue With Github
        </Button>
    </div>
}