"use client"

import { useState } from "react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/ui/components/alert-dialog"
import { Loader2 } from "lucide-react"
import { Input } from "@/ui/components/input"
import { Field, FieldLabel } from "@/ui/components/field"

export interface ConfirmDialogDetail {
  label: string
  value: string
}

interface ConfirmDialogBaseProps {
  title: string
  description: string
  /** Context details rendered as key-value pairs below the description */
  details?: ConfirmDialogDetail[]
  /** When set, user must type this exact text to enable the action button */
  confirmText?: string
  /** Custom label shown above the confirm input */
  confirmLabel?: string
  actionLabel?: string
  cancelLabel?: string
  onConfirm: () => void | Promise<void>
}

interface ConfirmDialogTriggerProps extends ConfirmDialogBaseProps {
  /** Trigger element - when provided, the dialog manages its own open state */
  children: React.ReactElement
  open?: never
  onOpenChange?: never
}

interface ConfirmDialogControlledProps extends ConfirmDialogBaseProps {
  /** Controlled mode - caller manages open state */
  open: boolean
  onOpenChange: (open: boolean) => void
  children?: never
}

type ConfirmDialogProps = ConfirmDialogTriggerProps | ConfirmDialogControlledProps

/**
 * ConfirmDialog - wraps AlertDialog to provide a confirm/cancel pattern
 * with context details, auto-focus on Cancel, and a spinner while processing.
 */
export const ConfirmDialog = (props: ConfirmDialogProps) => {
  const {
    title,
    description,
    details,
    confirmText,
    confirmLabel,
    actionLabel = "Confirm",
    cancelLabel = "Cancel",
    onConfirm,
  } = props

  // Internal state for trigger mode
  const [internalOpen, setInternalOpen] = useState(false)
  const [pending, setPending] = useState(false)
  const [confirmInput, setConfirmInput] = useState("")

  const isControlled = "open" in props && props.open !== undefined
  const open = isControlled ? props.open : internalOpen
  const setOpen = isControlled
    ? (o: boolean) => { if (!o) setConfirmInput(""); props.onOpenChange(o) }
    : (o: boolean) => { if (!o) setConfirmInput(""); setInternalOpen(o) }

  const requiresConfirmText = !!confirmText
  const confirmMatches = !requiresConfirmText || confirmInput === confirmText

  const handleConfirm = async () => {
    setPending(true)
    try {
      await onConfirm()
      setOpen(false)
    } finally {
      setPending(false)
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={(o) => setOpen(o)}>
      {"children" in props && props.children && (
        <AlertDialogTrigger render={props.children} />
      )}
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        {details && details.length > 0 && (
          <dl className="rounded-md bg-muted/50 p-3 text-sm space-y-1.5">
            {details.map((d) => (
              <div key={d.label} className="flex justify-between gap-4">
                <dt className="text-muted-foreground">{d.label}</dt>
                <dd className="font-medium text-right truncate">{d.value}</dd>
              </div>
            ))}
          </dl>
        )}
        {requiresConfirmText && (
          <Field>
            <FieldLabel>
              {confirmLabel ?? <>Type <span className="font-semibold text-foreground">{confirmText}</span> to confirm</>}
            </FieldLabel>
            <Input
              value={confirmInput}
              onChange={(e) => setConfirmInput(e.target.value)}
              placeholder={confirmText}
              autoComplete="off"
            />
          </Field>
        )}
        <AlertDialogFooter>
          {/* autoFocus on Cancel = safer default per design spec */}
          <AlertDialogCancel disabled={pending} autoFocus>
            {cancelLabel}
          </AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={pending || !confirmMatches}
            aria-label={`Confirm ${actionLabel.toLowerCase()}`}
          >
            {pending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {actionLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
