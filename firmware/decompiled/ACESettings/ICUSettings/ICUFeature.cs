using System;
using System.Collections.Generic;
using System.Xml;
using System.Xml.Linq;

namespace ICUSettings;

public class ICUFeature : ICUBaseObject
{
	private string m_sName;

	private string m_sID;

	private ICURights m_eDefault = ICURights.Full;

	private ICUFeatureType m_eType;

	private string m_sComment;

	public ICUFeature OriginalValues { get; set; }

	public string Name
	{
		get
		{
			return m_sName;
		}
		set
		{
			m_sName = value.Trim();
			FireChangedEvent("Name");
			CheckDirty();
		}
	}

	public string ID
	{
		get
		{
			return m_sID;
		}
		set
		{
			m_sID = value.Trim().ToUpperInvariant();
			FireChangedEvent("ID");
			CheckDirty();
		}
	}

	public ICURights Default
	{
		get
		{
			return m_eDefault;
		}
		set
		{
			m_eDefault = value;
			FireChangedEvent("Default");
			CheckDirty();
		}
	}

	public ICUFeatureType Type
	{
		get
		{
			return m_eType;
		}
		set
		{
			m_eType = value;
			FireChangedEvent("Type");
			CheckDirty();
		}
	}

	public string Comment
	{
		get
		{
			return m_sComment;
		}
		set
		{
			m_sComment = value.Trim();
			FireChangedEvent("Comment");
			CheckDirty();
		}
	}

	public XElement Element => new XElement("Feature", new XAttribute("Name", Name), new XAttribute("ID", ID), new XAttribute("Type", Type), new XAttribute("Default", Default), new XAttribute("Comment", Comment));

	public string Json => string.Format("\n\t{{\"Name\":{0},\"ID\":{1},\"Type\":{2},\"Default\":{3},\"Comment\":{4}}}", new object[5]
	{
		ValidString(Name),
		ValidString(ID),
		ValidString(Type.ToString()),
		ValidString(Default.ToString()),
		ValidString(Comment)
	});

	public ICUFeature()
		: base(fDirty: true)
	{
		m_sName = "";
		m_sID = "";
		m_eType = ICUFeatureType.Normal;
		m_eDefault = ICURights.None;
		m_sComment = "";
	}

	public ICUFeature(List<ICUGroup> allGroups, XElement xelem)
	{
		m_sName = getAttribute(xelem, "Name").Trim();
		m_sID = getAttribute(xelem, "ID").Trim();
		m_eType = (ICUFeatureType)Enum.Parse(typeof(ICUFeatureType), getAttribute(xelem, "Type").Trim());
		m_eDefault = (ICURights)Enum.Parse(typeof(ICURights), getAttribute(xelem, "Default").Trim());
		if (xelem.NextNode != null && xelem.NextNode.NodeType == XmlNodeType.Comment)
		{
			m_sComment = ((XComment)xelem.NextNode).Value.Trim();
		}
		OriginalValues = new ICUFeature(Name, ID, Type, Default, Comment);
	}

	public ICUFeature(dynamic obj)
	{
		m_sName = obj["Name"].Trim();
		m_sID = obj["ID"].Trim();
		if (((IDictionary<string, object>)obj).ContainsKey("Type"))
		{
			m_eType = (ICUFeatureType)Enum.Parse(typeof(ICUFeatureType), obj["Type"].Trim());
		}
		if (((IDictionary<string, object>)obj).ContainsKey("Default"))
		{
			m_eDefault = (ICURights)Enum.Parse(typeof(ICURights), obj["Default"].Trim());
		}
		m_sComment = obj["Comment"].Trim().Replace("<br>", "\n");
		OriginalValues = new ICUFeature(Name, ID, Type, Default, Comment);
	}

	public ICUFeature(string name, string id, ICUFeatureType eType, ICURights eDefaultRights = ICURights.Full, string comment = "")
	{
		m_sName = name.Trim();
		m_sID = id.Trim();
		m_eType = eType;
		m_eDefault = eDefaultRights;
		m_sComment = comment.Trim();
		OriginalValues = new ICUFeature();
		OriginalValues.Name = name;
		OriginalValues.ID = id;
		OriginalValues.Type = eType;
		OriginalValues.Default = eDefaultRights;
		OriginalValues.Comment = comment;
	}

	public static ICUFeature CreatePageFeature(string id, string name, ICURights eDefaultRights = ICURights.Full)
	{
		return new ICUFeature(name, $"PAGE_{id}", ICUFeatureType.Page, eDefaultRights);
	}

	public static ICUFeature CreateFeature(string id, string name, ICURights eDefaultRights = ICURights.Full)
	{
		return new ICUFeature(name, $"FEATURE_{id}", ICUFeatureType.Normal, eDefaultRights);
	}

	public static ICUFeature CreateIDFeature(ushort usId, string name, ICURights eDefaultRights = ICURights.Full)
	{
		return new ICUFeature(name, $"ID_{usId:X4}", ICUFeatureType.Property, eDefaultRights);
	}

	public static ICUFeature CreateBackOfficeFeature(string title, string name, ICURights eDefaultRights = ICURights.ReadOnly)
	{
		return new ICUFeature(name, $"BO_{title.Replace(' ', '_').Trim().ToUpperInvariant()}", ICUFeatureType.Backoffice, eDefaultRights);
	}

	protected override bool OnCheckDirty()
	{
		if (OriginalValues == null)
		{
			return true;
		}
		if (!(m_sName != OriginalValues.Name) && !(m_sID != OriginalValues.ID) && m_eType == OriginalValues.Type)
		{
			return m_sComment != OriginalValues.Comment;
		}
		return true;
	}

	public void CopyFrom(ICUFeature other)
	{
		if (other != null)
		{
			Name = other.Name;
			ID = other.ID;
			Type = other.Type;
			Default = other.Default;
			Comment = other.Comment;
			CheckDirty();
		}
	}

	public void Commit()
	{
		if (OriginalValues != null)
		{
			OriginalValues.CopyFrom(this);
		}
		CheckDirty();
	}

	public void Rollback()
	{
		if (OriginalValues != null)
		{
			CopyFrom(OriginalValues);
		}
		CheckDirty();
	}
}
