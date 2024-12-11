import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Queue, QueueTaskStatus, useEditQueue } from "@/app/hooks/useQueue";
import { Button, TextInput, Checkbox, Select } from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { useState } from "react";
import { z } from "zod";

type Props = {
  queue: Queue
  handleClose: () => void;
}


const schema = z.object({
  id: z.string().min(2, { message: "ID should have at least 2 characters" }),
  processing: z.boolean(),
  on_hold: z.boolean(),
  video_processing: z.boolean(),
  chat_processing: z.boolean(),
  live_archive: z.boolean(),

  task_vod_create_folder: z.nativeEnum(QueueTaskStatus),
  task_vod_download_thumbnail: z.nativeEnum(QueueTaskStatus),
  task_vod_save_info: z.nativeEnum(QueueTaskStatus),
  task_video_download: z.nativeEnum(QueueTaskStatus),
  task_video_convert: z.nativeEnum(QueueTaskStatus),
  task_video_move: z.nativeEnum(QueueTaskStatus),
  task_chat_convert: z.nativeEnum(QueueTaskStatus),
  task_chat_render: z.nativeEnum(QueueTaskStatus),
  task_chat_move: z.nativeEnum(QueueTaskStatus),
})

const AdminQueueDrawerContent = ({ queue, handleClose }: Props) => {
  const axiosPrivate = useAxiosPrivate()
  const useEditQueueMutate = useEditQueue()
  const [loading, setLoading] = useState(false)

  const form = useForm({
    mode: "controlled",
    initialValues: {
      id: queue.id,
      processing: queue.processing ?? false,
      on_hold: queue.on_hold ?? false,
      video_processing: queue.video_processing ?? false,
      chat_processing: queue.chat_processing ?? false,
      live_archive: queue.live_archive ?? false,

      task_vod_create_folder: queue.task_vod_create_folder,
      task_vod_download_thumbnail: queue.task_vod_download_thumbnail,
      task_vod_save_info: queue.task_vod_save_info,
      task_video_download: queue.task_video_download,
      task_video_convert: queue.task_video_convert,
      task_video_move: queue.task_video_move,
      task_chat_download: queue.task_chat_download,
      task_chat_convert: queue.task_chat_convert,
      task_chat_render: queue.task_chat_render,
      task_chat_move: queue.task_chat_move,
    },

    validate: zodResolver(schema),
  })

  const handleSubmitForm = async () => {
    const formValues = form.getValues()
    console.debug(`Queue form submit - ${formValues}`)

    // @ts-expect-error uncessary
    const submitQueue: Queue = { ...formValues }

    // edit queue
    try {
      setLoading(true)

      await useEditQueueMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        queue: submitQueue
      })

      showNotification({
        message: "Queue edited"
      })

      handleClose()
    } catch (error) {
      setLoading(false)
      console.error(error)
    }
  }

  const selectorQueueTaskStatus = Object.values(QueueTaskStatus).map((type) => ({
    value: type,
    label: type.charAt(0).toUpperCase() + type.slice(1),
  }));

  return (
    <div>
      <form onSubmit={form.onSubmit(() => {
        handleSubmitForm()
      })}>
        <TextInput
          disabled={true}
          label="ID"
          placeholder="Auto generated"
          key={form.key('id')}
          {...form.getInputProps('id')}
        />

        <Checkbox
          label="Is Processing"
          key={form.key('processing')}
          {...form.getInputProps('processing', { type: "checkbox" })}
        />
        <Checkbox
          label="On Hold"
          key={form.key('on_hold')}
          {...form.getInputProps('on_hold', { type: "checkbox" })}
        />
        <Checkbox
          label="Video Processing"
          key={form.key('video_processing')}
          {...form.getInputProps('video_processing', { type: "checkbox" })}
        />
        <Checkbox
          label="Chat Processing"
          key={form.key('chat_processing')}
          {...form.getInputProps('chat_processing', { type: "checkbox" })}
        />
        <Checkbox
          label="Live Archive"
          key={form.key('live_archive')}
          {...form.getInputProps('live_archive', { type: "checkbox" })}
        />

        <Select
          label="Task Create Folder"
          data={selectorQueueTaskStatus}
          key={form.key('task_vod_create_folder')}
          {...form.getInputProps('task_vod_create_folder')}
        />
        <Select
          label="Task Download Thumbnail"
          data={selectorQueueTaskStatus}
          key={form.key('task_vod_download_thumbnail')}
          {...form.getInputProps('task_vod_download_thumbnail')}
        />
        <Select
          label="Task Save Information"
          data={selectorQueueTaskStatus}
          key={form.key('task_vod_save_info')}
          {...form.getInputProps('task_vod_save_info')}
        />
        <Select
          label="Task Video Download"
          data={selectorQueueTaskStatus}
          key={form.key('task_video_download')}
          {...form.getInputProps('task_video_download')}
        />
        <Select
          label="Task Video Convert"
          data={selectorQueueTaskStatus}
          key={form.key('task_video_convert')}
          {...form.getInputProps('task_video_convert')}
        />
        <Select
          label="Task Video Move"
          data={selectorQueueTaskStatus}
          key={form.key('task_video_move')}
          {...form.getInputProps('task_video_move')}
        />
        <Select
          label="Task Chat Download"
          data={selectorQueueTaskStatus}
          key={form.key('task_chat_download')}
          {...form.getInputProps('task_chat_download')}
        />
        <Select
          label="Task Chat Convert"
          data={selectorQueueTaskStatus}
          key={form.key('task_chat_convert')}
          {...form.getInputProps('task_chat_convert')}
        />
        <Select
          label="Task Chat Render"
          data={selectorQueueTaskStatus}
          key={form.key('task_chat_render')}
          {...form.getInputProps('task_chat_render')}
        />
        <Select
          label="Task Chat Move"
          data={selectorQueueTaskStatus}
          key={form.key('task_chat_move')}
          {...form.getInputProps('task_chat_move')}
        />

        <Button mt={10} type="submit" loading={loading} fullWidth>Edit Queue Item</Button>
      </form>

    </div>
  );
}

export default AdminQueueDrawerContent;