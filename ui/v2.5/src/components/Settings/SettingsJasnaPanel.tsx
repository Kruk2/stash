import React, { useRef, useState } from "react";
import { Button, Form } from "react-bootstrap";
import { useIntl } from "react-intl";
import { SettingSection } from "./SettingSection";
import { useSettings } from "./context";
import * as GQL from "src/core/generated-graphql";
import { SettingModal } from "./Inputs";
import { LoadingIndicator } from "../Shared/LoadingIndicator";

export interface IJasnaModal {
  value: GQL.JasnaPresetInput;
  close: (v?: GQL.JasnaPresetInput) => void;
}

const defaultMaxClipSize = 180;
const defaultTemporalOverlap = 10;

export const JasnaModal: React.FC<IJasnaModal> = ({ value, close }) => {
  const intl = useIntl();
  const nameRef = useRef<HTMLInputElement | null>(null);

  return (
    <SettingModal<GQL.JasnaPresetInput>
      headingID="config.jasna.preset_title"
      value={value}
      renderField={(v, setValue) => (
        <>
          <Form.Group id="jasna-preset-name">
            <h6>
              {intl.formatMessage({ id: "config.jasna.preset_name" })}
            </h6>
            <Form.Control
              placeholder={intl.formatMessage({ id: "config.jasna.preset_name" })}
              className="text-input jasna-preset-name"
              value={v?.name}
              isValid={(v?.name?.length ?? 0) > 0}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setValue({ ...v!, name: e.currentTarget.value })
              }
              ref={nameRef}
            />
          </Form.Group>

          <Form.Group id="jasna-max-clip-size">
            <h6>
              {intl.formatMessage({ id: "config.jasna.max_clip_size" })}
            </h6>
            <Form.Control
              placeholder={intl.formatMessage({ id: "config.jasna.max_clip_size" })}
              className="text-input"
              value={v?.maxClipSize ?? defaultMaxClipSize}
              isValid={(v?.maxClipSize ?? defaultMaxClipSize) > 0}
              type="number"
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setValue({
                  ...v!,
                  maxClipSize: parseInt(e.currentTarget.value),
                })
              }
            />
            <div className="sub-heading">
              {intl.formatMessage({ id: "config.jasna.max_clip_size_description" })}
            </div>
          </Form.Group>

          <Form.Group id="jasna-temporal-overlap">
            <h6>
              {intl.formatMessage({ id: "config.jasna.temporal_overlap" })}
            </h6>
            <Form.Control
              placeholder={intl.formatMessage({ id: "config.jasna.temporal_overlap" })}
              className="text-input"
              value={v?.temporalOverlap ?? defaultTemporalOverlap}
              isValid={(v?.temporalOverlap ?? defaultTemporalOverlap) >= 0}
              type="number"
              onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                setValue({
                  ...v!,
                  temporalOverlap: parseInt(e.currentTarget.value),
                })
              }
            />
            <div className="sub-heading">
              {intl.formatMessage({ id: "config.jasna.temporal_overlap_description" })}
            </div>
          </Form.Group>

          <Form.Group id="jasna-secondary-restoration">
            <h6>
              {intl.formatMessage({ id: "config.jasna.secondary_restoration" })}
            </h6>
            <Form.Control
              as="select"
              className="text-input"
              value={v?.secondaryRestoration ?? ""}
              onChange={(e: React.ChangeEvent<HTMLSelectElement>) =>
                setValue({
                  ...v!,
                  secondaryRestoration: e.currentTarget.value || undefined,
                })
              }
            >
              <option value="">
                {intl.formatMessage({ id: "config.jasna.restoration.none" })}
              </option>
              <option value="unet-4x">
                {intl.formatMessage({ id: "config.jasna.restoration.unet_4x" })}
              </option>
              <option value="rtx-super-res">
                {intl.formatMessage({ id: "config.jasna.restoration.rtx_super_res" })}
              </option>
            </Form.Control>
          </Form.Group>
        </>
      )}
      close={close}
    />
  );
};

interface IJasnaSetting {
  value: GQL.JasnaPresetInput[];
  onChange: (v: GQL.JasnaPresetInput[]) => void;
}

export const JasnaSetting: React.FC<IJasnaSetting> = ({ value, onChange }) => {
  const [isCreating, setIsCreating] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | undefined>();

  function onEdit(index: number) {
    setEditingIndex(index);
  }

  function onDelete(index: number) {
    onChange(value.filter((v, i) => i !== index));
  }

  function onNew() {
    setIsCreating(true);
  }

  return (
    <SettingSection
      id="jasna-presets"
      headingID="config.jasna.presets"
      subHeadingID="config.jasna.presets_description"
    >
      {isCreating ? (
        <JasnaModal
          value={{
            name: "",
            maxClipSize: defaultMaxClipSize,
            temporalOverlap: defaultTemporalOverlap,
            secondaryRestoration: "",
          }}
          close={(v) => {
            if (v) onChange([...value, v]);
            setIsCreating(false);
          }}
        />
      ) : undefined}

      {editingIndex !== undefined ? (
        <JasnaModal
          value={value[editingIndex]}
          close={(v) => {
            if (v)
              onChange(
                value.map((vv, index) => {
                  if (index === editingIndex) {
                    return v;
                  }
                  return vv;
                })
              );
            setEditingIndex(undefined);
          }}
        />
      ) : undefined}

      {value.map((preset, index) => (
        // eslint-disable-next-line react/no-array-index-key
        <div key={index} className="setting">
          <div>
            <h3>{preset.name ?? `#${index}`}</h3>
            <div className="value">
              {preset.maxClipSize}s clip, {preset.temporalOverlap}s overlap
              {preset.secondaryRestoration ? (
                <>
                  ,{" "}
                  {preset.secondaryRestoration === "unet-4x"
                    ? "UNet 4x"
                    : preset.secondaryRestoration === "rtx-super-res"
                    ? "RTX Super Resolution"
                    : preset.secondaryRestoration}
                </>
              ) : null}
            </div>
          </div>
          <div>
            <Button onClick={() => onEdit(index)}>
              Edit
            </Button>
            <Button variant="danger" onClick={() => onDelete(index)}>
              Delete
            </Button>
          </div>
        </div>
      ))}
      <div className="setting">
        <div />
        <div>
          <Button onClick={() => onNew()}>
            Add Preset
          </Button>
        </div>
      </div>
    </SettingSection>
  );
};

export const SettingsJasnaPanel: React.FC = () => {
  const { jasna, loading, error, saveJasna } = useSettings();

  if (error) return <h1>{error.message}</h1>;
  if (loading) return <LoadingIndicator />;

  return (
    <JasnaSetting
      value={jasna.presets ?? []}
      onChange={(v) => saveJasna({ presets: v })}
    />
  );
};
